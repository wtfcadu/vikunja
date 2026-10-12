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

package apiv2

import (
	"context"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
)

// RegisterProjectFinanceRoutes wires the owner-only finance details, payments
// and the projects overview onto the Huma API.
func RegisterProjectFinanceRoutes(api huma.API) {
	tags := []string{"project_finance"}

	Register(api, huma.Operation{
		OperationID: "project-finance-read",
		Summary:     "Get the finance details of a project",
		Description: "Returns the client, currency and agreed fee of a project. Only the project owner may read them; everyone else gets 403. A project without saved details returns empty details.",
		Method:      http.MethodGet,
		Path:        "/projects/{project}/finance",
		Tags:        tags,
	}, projectFinanceRead)

	Register(api, huma.Operation{
		OperationID: "project-finance-update",
		Summary:     "Save the finance details of a project",
		Description: "Replaces the client, currency and agreed fee of a project, creating them on first save. Only the project owner may change them. The currency must be a three-letter code; it is stored upper-case.",
		Method:      http.MethodPut,
		Path:        "/projects/{project}/finance",
		Tags:        tags,
	}, projectFinanceUpdate)

	Register(api, huma.Operation{
		OperationID: "project-payments-list",
		Summary:     "List the payments of a project",
		Description: "Returns the recorded payments of a project, newest first. Only the project owner may list them.",
		Method:      http.MethodGet,
		Path:        "/projects/{project}/payments",
		Tags:        tags,
	}, projectPaymentsList)

	Register(api, huma.Operation{
		OperationID: "project-payments-read",
		Summary:     "Get a payment of a project",
		Description: "Returns one payment. The payment must belong to the project in the path. Only the project owner may read it.",
		Method:      http.MethodGet,
		Path:        "/projects/{project}/payments/{payment}",
		Tags:        tags,
	}, projectPaymentsRead)

	Register(api, huma.Operation{
		OperationID: "project-payments-create",
		Summary:     "Record a payment for a project",
		Description: "Records a payment in the project's currency. A missing paid_at defaults to now. Only the project owner may record payments.",
		Method:      http.MethodPost,
		Path:        "/projects/{project}/payments",
		Tags:        tags,
	}, projectPaymentsCreate)

	Register(api, huma.Operation{
		OperationID: "project-payments-update",
		Summary:     "Update a payment of a project",
		Description: "Replaces the amount, date and note of a payment. Only the project owner may change it.",
		Method:      http.MethodPut,
		Path:        "/projects/{project}/payments/{payment}",
		Tags:        tags,
	}, projectPaymentsUpdate)

	Register(api, huma.Operation{
		OperationID: "project-payments-delete",
		Summary:     "Delete a payment of a project",
		Description: "Deletes a payment. Only the project owner may delete it.",
		Method:      http.MethodDelete,
		Path:        "/projects/{project}/payments/{payment}",
		Tags:        tags,
	}, projectPaymentsDelete)

	Register(api, huma.Operation{
		OperationID: "project-overview-list",
		Summary:     "Overview of the projects you own",
		Description: "Returns every project the authenticated user owns, archived ones included, with client, fee, paid and owed amounts, task counts and the people assigned to tasks. Link shares get 403. Not paginated.",
		Method:      http.MethodGet,
		Path:        "/project-overview",
		Tags:        tags,
	}, projectOverviewList)
}

func init() { AddRouteRegistrar(RegisterProjectFinanceRoutes) }

type projectFinanceReadBody struct {
	models.ProjectFinance
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The maximum permission the requesting user has on these details (always 2 for the owner)."`
}

func projectFinanceRead(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	conditional.Params
}) (*singleReadBody[projectFinanceReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	pf := &models.ProjectFinance{ProjectID: in.ProjectID}
	maxPermission, err := handler.DoReadOne(ctx, pf, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &projectFinanceReadBody{ProjectFinance: *pf, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, pf.Updated, maxPermission)
}

func projectFinanceUpdate(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	Body      projectFinanceReadBody
}) (*singleBody[models.ProjectFinance], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	pf := &in.Body.ProjectFinance
	pf.ProjectID = in.ProjectID // URL wins over body
	if err := handler.DoUpdate(ctx, pf, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.ProjectFinance]{Body: pf}, nil
}

type projectPaymentListBody struct {
	Body Paginated[*models.ProjectPayment]
}

func projectPaymentsList(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	ListParams
}) (*projectPaymentListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &models.ProjectPayment{ProjectID: in.ProjectID}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.ProjectPayment)
	if !ok {
		return nil, fmt.Errorf("projectPayments.ReadAll returned unexpected type %T (expected []*models.ProjectPayment)", result)
	}
	return &projectPaymentListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

type projectPaymentReadBody struct {
	models.ProjectPayment
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The maximum permission the requesting user has on this payment (always 2 for the owner)."`
}

func projectPaymentsRead(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	ID        int64 `path:"payment"`
	conditional.Params
}) (*singleReadBody[projectPaymentReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	pp := &models.ProjectPayment{ID: in.ID, ProjectID: in.ProjectID}
	maxPermission, err := handler.DoReadOne(ctx, pp, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &projectPaymentReadBody{ProjectPayment: *pp, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, pp.Updated, maxPermission)
}

func projectPaymentsCreate(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	Body      models.ProjectPayment
}) (*singleBody[models.ProjectPayment], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	in.Body.ProjectID = in.ProjectID // URL wins over body
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.ProjectPayment]{Body: &in.Body}, nil
}

func projectPaymentsUpdate(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	ID        int64 `path:"payment"`
	Body      projectPaymentReadBody
}) (*singleBody[models.ProjectPayment], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	pp := &in.Body.ProjectPayment
	pp.ID = in.ID
	pp.ProjectID = in.ProjectID
	if err := handler.DoUpdate(ctx, pp, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.ProjectPayment]{Body: pp}, nil
}

func projectPaymentsDelete(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
	ID        int64 `path:"payment"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.ProjectPayment{ID: in.ID, ProjectID: in.ProjectID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

type projectOverview struct {
	Items []*models.ProjectOverviewItem `json:"items" doc:"Every project the requesting user owns."`
}

type projectOverviewBody struct {
	Body *projectOverview
}

func projectOverviewList(ctx context.Context, _ *struct{}) (*projectOverviewBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	u, isUser := a.(*user.User)
	if !isUser {
		return nil, huma.Error403Forbidden("only users can see the projects overview")
	}
	s := db.NewSession()
	defer s.Close()
	items, err := models.GetProjectOverview(s, u)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &projectOverviewBody{Body: &projectOverview{Items: items}}, nil
}
