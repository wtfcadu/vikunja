// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"sort"

	"code.vikunja.io/api/pkg/user"

	"xorm.io/xorm"
)

// ProjectOverviewWorker is a person with tasks on a project.
type ProjectOverviewWorker struct {
	ID        int64  `json:"id" doc:"The user's id."`
	Username  string `json:"username" doc:"The user's username."`
	Name      string `json:"name" doc:"The user's full name, if set."`
	OpenTasks int64  `json:"open_tasks" doc:"How many unfinished tasks on the project are assigned to this person."`
}

// ProjectOverviewItem is one row of the projects overview: the project, its
// client and money, its task progress and the people working on it.
type ProjectOverviewItem struct {
	ID              int64  `json:"id" doc:"The project id."`
	Title           string `json:"title" doc:"The project title."`
	HexColor        string `json:"hex_color" doc:"The project color without the leading #."`
	IsArchived      bool   `json:"is_archived" doc:"Whether the project is archived."`
	ParentProjectID int64  `json:"parent_project_id" doc:"The parent project id, or 0 for a top-level project."`

	Client   string `json:"client" doc:"The client the project is for."`
	Currency string `json:"currency" doc:"ISO 4217 currency code of the fee and payments."`
	Fee      int64  `json:"fee" doc:"The agreed fee in minor units."`
	Paid     int64  `json:"paid" doc:"The sum of all recorded payments in minor units."`
	Owed     int64  `json:"owed" doc:"Fee minus paid in minor units, never below zero."`
	Payments int64  `json:"payments" doc:"How many payments were recorded."`

	OpenTasks   int64 `json:"open_tasks" doc:"Unfinished tasks on the project."`
	DoneTasks   int64 `json:"done_tasks" doc:"Finished tasks on the project."`
	MyOpenTasks int64 `json:"my_open_tasks" doc:"Unfinished tasks on the project assigned to the requesting user."`

	Workers []*ProjectOverviewWorker `json:"workers" doc:"Everyone with at least one task on the project assigned to them, the requesting user included."`
}

type overviewTaskRow struct {
	ID        int64 `xorm:"id"`
	ProjectID int64 `xorm:"project_id"`
	Done      bool  `xorm:"done"`
}

type overviewAssigneeRow struct {
	UserID    int64 `xorm:"user_id"`
	ProjectID int64 `xorm:"project_id"`
	Done      bool  `xorm:"done"`
}

type overviewPaymentRow struct {
	ProjectID int64 `xorm:"project_id"`
	Amount    int64 `xorm:"amount"`
}

// GetProjectOverview returns every project the user owns, archived ones
// included, with its finance details, payment totals, task counts and workers.
// Only owned projects are returned, so the money of a project is only ever
// shown to its owner.
func GetProjectOverview(s *xorm.Session, u *user.User) ([]*ProjectOverviewItem, error) {
	projects := []*Project{}
	if err := s.Where("owner_id = ?", u.ID).OrderBy("position, id").Find(&projects); err != nil {
		return nil, err
	}
	items := make([]*ProjectOverviewItem, 0, len(projects))
	if len(projects) == 0 {
		return items, nil
	}

	ids := make([]int64, 0, len(projects))
	byID := make(map[int64]*ProjectOverviewItem, len(projects))
	for _, p := range projects {
		item := &ProjectOverviewItem{
			ID:              p.ID,
			Title:           p.Title,
			HexColor:        p.HexColor,
			IsArchived:      p.IsArchived,
			ParentProjectID: p.parentID(),
			Workers:         []*ProjectOverviewWorker{},
		}
		items = append(items, item)
		byID[p.ID] = item
		ids = append(ids, p.ID)
	}

	finances := []*ProjectFinance{}
	if err := s.In("project_id", ids).Find(&finances); err != nil {
		return nil, err
	}
	for _, f := range finances {
		if item, ok := byID[f.ProjectID]; ok {
			item.Client = f.Client
			item.Currency = f.Currency
			item.Fee = f.Fee
		}
	}

	payments := []*overviewPaymentRow{}
	if err := s.Table("project_payments").Select("project_id, amount").In("project_id", ids).Find(&payments); err != nil {
		return nil, err
	}
	for _, p := range payments {
		if item, ok := byID[p.ProjectID]; ok {
			item.Paid += p.Amount
			item.Payments++
		}
	}

	tasks := []*overviewTaskRow{}
	if err := s.Table("tasks").Select("id, project_id, done").In("project_id", ids).Find(&tasks); err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if item, ok := byID[t.ProjectID]; ok {
			if t.Done {
				item.DoneTasks++
			} else {
				item.OpenTasks++
			}
		}
	}

	assignees := []*overviewAssigneeRow{}
	if err := s.Table("task_assignees").
		Join("INNER", "tasks", "tasks.id = task_assignees.task_id").
		Select("task_assignees.user_id AS user_id, tasks.project_id AS project_id, tasks.done AS done").
		In("tasks.project_id", ids).
		Find(&assignees); err != nil {
		return nil, err
	}

	userIDs := []int64{}
	seenUser := map[int64]bool{}
	workers := map[int64]map[int64]*ProjectOverviewWorker{} // project -> user -> worker
	for _, a := range assignees {
		item, ok := byID[a.ProjectID]
		if !ok {
			continue
		}
		if !seenUser[a.UserID] {
			seenUser[a.UserID] = true
			userIDs = append(userIDs, a.UserID)
		}
		if workers[a.ProjectID] == nil {
			workers[a.ProjectID] = map[int64]*ProjectOverviewWorker{}
		}
		w, ok := workers[a.ProjectID][a.UserID]
		if !ok {
			w = &ProjectOverviewWorker{ID: a.UserID}
			workers[a.ProjectID][a.UserID] = w
			item.Workers = append(item.Workers, w)
		}
		if !a.Done {
			w.OpenTasks++
			if a.UserID == u.ID {
				item.MyOpenTasks++
			}
		}
	}

	if len(userIDs) > 0 {
		users, err := user.GetUsersByIDs(s, userIDs)
		if err != nil {
			return nil, err
		}
		for _, perProject := range workers {
			for id, w := range perProject {
				if usr, ok := users[id]; ok {
					w.Username = usr.Username
					w.Name = usr.Name
				}
			}
		}
	}

	for _, item := range items {
		item.Owed = max(item.Fee-item.Paid, 0)
		sort.Slice(item.Workers, func(i, j int) bool {
			return item.Workers[i].Username < item.Workers[j].Username
		})
	}

	return items, nil
}
