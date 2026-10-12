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

package migration

import (
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

type projectFinance20261012140000 struct {
	ProjectID int64     `xorm:"bigint not null unique pk"`
	Client    string    `xorm:"varchar(250) null"`
	Currency  string    `xorm:"varchar(3) null"`
	Fee       int64     `xorm:"bigint not null default 0"`
	Created   time.Time `xorm:"created not null"`
	Updated   time.Time `xorm:"updated not null"`
}

func (projectFinance20261012140000) TableName() string { return "project_finances" }

type projectPayment20261012140000 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	ProjectID   int64     `xorm:"bigint not null index"`
	Amount      int64     `xorm:"bigint not null default 0"`
	PaidAt      time.Time `xorm:"DATETIME not null"`
	Note        string    `xorm:"varchar(250) null"`
	CreatedByID int64     `xorm:"bigint not null"`
	Created     time.Time `xorm:"created not null"`
	Updated     time.Time `xorm:"updated not null"`
}

func (projectPayment20261012140000) TableName() string { return "project_payments" }

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261012140000",
		Description: "Add project finance details and payments",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync(projectFinance20261012140000{}, projectPayment20261012140000{}) //nolint:forbidigo // brand-new tables, nothing to drop
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(projectPayment20261012140000{}, projectFinance20261012140000{})
		},
	})
}
