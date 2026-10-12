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
	"strings"
	"time"

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// ProjectFinance holds the business details of a project: who the client is,
// the currency and the agreed fee. Only the project owner can see or change it,
// so people who only work on the project never see the money.
type ProjectFinance struct {
	ProjectID int64 `xorm:"bigint not null unique pk" json:"project_id" param:"project" readOnly:"true" doc:"The project these details belong to. Taken from the URL path; ignored on write."`
	// The client the project is for.
	Client string `xorm:"varchar(250) null" json:"client" valid:"runelength(0|250)" maxLength:"250" doc:"The client the project is for. Free text."`
	// The ISO 4217 currency code of the fee and all payments, e.g. GBP.
	Currency string `xorm:"varchar(3) null" json:"currency" valid:"runelength(0|3)" maxLength:"3" doc:"ISO 4217 currency code of the fee and of every payment on this project, e.g. GBP. Stored upper-case."`
	// The agreed fee in minor units (cents, pence).
	Fee int64 `xorm:"bigint not null default 0" json:"fee" minimum:"0" doc:"The agreed fee for the project in minor units of the currency (pence, cents)."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"A timestamp when these details were first saved."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"A timestamp when these details were last changed."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName returns the table name for project finances
func (*ProjectFinance) TableName() string {
	return "project_finances"
}

// isProjectOwner reports whether the auth is the owner of the project. Link
// shares and anyone other than the owner are refused. A missing project is an
// error so the caller gets a 404 instead of a 403.
func isProjectOwner(s *xorm.Session, a web.Auth, projectID int64) (bool, error) {
	if IsPseudoProjectID(projectID) {
		return false, ErrProjectDoesNotExist{ID: projectID}
	}
	project, err := GetProjectSimpleByID(s, projectID)
	if err != nil {
		return false, err
	}
	u, isUser := a.(*user.User)
	if !isUser {
		return false, nil
	}
	return project.OwnerID == u.ID, nil
}

// CanRead allows only the project owner to read the finance details.
func (pf *ProjectFinance) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	is, err := isProjectOwner(s, a, pf.ProjectID)
	if err != nil || !is {
		return false, 0, err
	}
	return true, int(PermissionAdmin), nil
}

// CanUpdate allows only the project owner to change the finance details.
func (pf *ProjectFinance) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return isProjectOwner(s, a, pf.ProjectID)
}

// CanCreate allows only the project owner to save the finance details.
func (pf *ProjectFinance) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	return isProjectOwner(s, a, pf.ProjectID)
}

// CanDelete allows only the project owner to clear the finance details.
func (pf *ProjectFinance) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return isProjectOwner(s, a, pf.ProjectID)
}

// getProjectFinance returns the stored details of a project, or empty details
// when none were saved yet.
func getProjectFinance(s *xorm.Session, projectID int64) (*ProjectFinance, bool, error) {
	pf := &ProjectFinance{}
	exists, err := s.Where("project_id = ?", projectID).Get(pf)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		pf = &ProjectFinance{ProjectID: projectID}
	}
	return pf, exists, nil
}

// ReadOne returns the finance details of a project. A project without saved
// details returns empty details rather than a 404.
// @Summary Get the finance details of a project
// @tags project
// @Produce json
// @Security JWTKeyAuth
// @Param project path int true "Project ID"
// @Success 200 {object} models.ProjectFinance
// @Failure 403 {object} web.HTTPError "Only the project owner can see this."
// @Router /projects/{project}/finance [get]
func (pf *ProjectFinance) ReadOne(s *xorm.Session, _ web.Auth) error {
	stored, _, err := getProjectFinance(s, pf.ProjectID)
	if err != nil {
		return err
	}
	*pf = *stored
	return nil
}

func (pf *ProjectFinance) normalize() {
	pf.Client = strings.TrimSpace(pf.Client)
	pf.Currency = strings.ToUpper(strings.TrimSpace(pf.Currency))
	if pf.Fee < 0 {
		pf.Fee = 0
	}
}

// Update saves the finance details of a project, creating them if needed.
// @Summary Save the finance details of a project
// @tags project
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param project path int true "Project ID"
// @Param details body models.ProjectFinance true "The finance details."
// @Success 200 {object} models.ProjectFinance
// @Failure 403 {object} web.HTTPError "Only the project owner can change this."
// @Router /projects/{project}/finance [post]
func (pf *ProjectFinance) Update(s *xorm.Session, _ web.Auth) error {
	pf.normalize()
	if err := validateCurrency(pf.Currency); err != nil {
		return err
	}
	_, exists, err := getProjectFinance(s, pf.ProjectID)
	if err != nil {
		return err
	}
	if !exists {
		if _, err := s.Insert(pf); err != nil {
			return err
		}
	} else {
		if _, err := s.Where("project_id = ?", pf.ProjectID).
			Cols("client", "currency", "fee").
			Update(pf); err != nil {
			return err
		}
	}
	stored, _, err := getProjectFinance(s, pf.ProjectID)
	if err != nil {
		return err
	}
	*pf = *stored
	return nil
}

// Create is the same as Update: the details are one row per project.
func (pf *ProjectFinance) Create(s *xorm.Session, a web.Auth) error {
	return pf.Update(s, a)
}

// Delete clears the finance details of a project. Payments are kept.
func (pf *ProjectFinance) Delete(s *xorm.Session, _ web.Auth) error {
	_, err := s.Where("project_id = ?", pf.ProjectID).Delete(&ProjectFinance{})
	return err
}

func validateCurrency(currency string) error {
	if currency == "" {
		return nil
	}
	if len(currency) != 3 {
		return &ErrInvalidCurrency{Currency: currency}
	}
	for _, r := range currency {
		if r < 'A' || r > 'Z' {
			return &ErrInvalidCurrency{Currency: currency}
		}
	}
	return nil
}

// ProjectPayment is one payment a client made towards a project. Amounts are
// in the currency set in the project's finance details.
type ProjectPayment struct {
	ID        int64 `xorm:"bigint autoincr not null unique pk" json:"id" param:"payment" readOnly:"true" doc:"The unique, numeric id of this payment."`
	ProjectID int64 `xorm:"bigint not null index" json:"project_id" param:"project" readOnly:"true" doc:"The project this payment belongs to. Taken from the URL path; ignored on write."`
	// The amount paid in minor units (cents, pence).
	Amount int64 `xorm:"bigint not null default 0" json:"amount" minimum:"0" doc:"The amount paid in minor units of the project's currency (pence, cents)."`
	// When the payment arrived.
	PaidAt time.Time `xorm:"DATETIME not null" json:"paid_at" doc:"When the payment arrived."`
	// A short note, e.g. "Deposit" or an invoice number.
	Note string `xorm:"varchar(250) null" json:"note" valid:"runelength(0|250)" maxLength:"250" doc:"A short note, such as 'Deposit' or an invoice number."`

	CreatedByID int64 `xorm:"bigint not null" json:"-"`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"A timestamp when this payment was recorded."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"A timestamp when this payment was last changed."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName returns the table name for project payments
func (*ProjectPayment) TableName() string {
	return "project_payments"
}

// loadIntoSelf loads the stored payment and checks it belongs to the project in
// the path, so a payment id from another project can't be reached through a
// project the caller owns.
func (pp *ProjectPayment) loadIntoSelf(s *xorm.Session) error {
	stored := &ProjectPayment{}
	exists, err := s.Where("id = ? AND project_id = ?", pp.ID, pp.ProjectID).Get(stored)
	if err != nil {
		return err
	}
	if !exists {
		return &ErrProjectPaymentDoesNotExist{PaymentID: pp.ID}
	}
	return nil
}

func (pp *ProjectPayment) canChange(s *xorm.Session, a web.Auth) (bool, error) {
	is, err := isProjectOwner(s, a, pp.ProjectID)
	if err != nil || !is {
		return false, err
	}
	if err := pp.loadIntoSelf(s); err != nil {
		return false, err
	}
	return true, nil
}

// CanRead allows only the project owner to see a payment.
func (pp *ProjectPayment) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	is, err := isProjectOwner(s, a, pp.ProjectID)
	if err != nil || !is {
		return false, 0, err
	}
	if pp.ID != 0 {
		if err := pp.loadIntoSelf(s); err != nil {
			return false, 0, err
		}
	}
	return true, int(PermissionAdmin), nil
}

// CanCreate allows only the project owner to record a payment.
func (pp *ProjectPayment) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	return isProjectOwner(s, a, pp.ProjectID)
}

// CanUpdate allows only the project owner to change a payment.
func (pp *ProjectPayment) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return pp.canChange(s, a)
}

// CanDelete allows only the project owner to delete a payment.
func (pp *ProjectPayment) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return pp.canChange(s, a)
}

// ReadAll returns every payment of a project, newest first.
// @Summary List the payments of a project
// @tags project
// @Produce json
// @Security JWTKeyAuth
// @Param project path int true "Project ID"
// @Success 200 {array} models.ProjectPayment
// @Router /projects/{project}/payments [get]
func (pp *ProjectPayment) ReadAll(s *xorm.Session, _ web.Auth, _ string, page int, perPage int) (result interface{}, resultCount int, numberOfTotalItems int64, err error) {
	limit, start := getLimitFromPageIndex(page, perPage)
	payments := []*ProjectPayment{}
	query := s.Where("project_id = ?", pp.ProjectID).OrderBy("paid_at DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit, start)
	}
	if err := query.Find(&payments); err != nil {
		return nil, 0, 0, err
	}
	total, err := s.Where("project_id = ?", pp.ProjectID).Count(&ProjectPayment{})
	if err != nil {
		return nil, 0, 0, err
	}
	return payments, len(payments), total, nil
}

// ReadOne returns one payment.
func (pp *ProjectPayment) ReadOne(s *xorm.Session, _ web.Auth) error {
	exists, err := s.Where("id = ? AND project_id = ?", pp.ID, pp.ProjectID).Get(pp)
	if err != nil {
		return err
	}
	if !exists {
		return &ErrProjectPaymentDoesNotExist{PaymentID: pp.ID}
	}
	return nil
}

func (pp *ProjectPayment) normalize() {
	pp.Note = strings.TrimSpace(pp.Note)
	if pp.Amount < 0 {
		pp.Amount = 0
	}
	if pp.PaidAt.IsZero() {
		pp.PaidAt = time.Now()
	}
}

// Create records a payment.
// @Summary Record a payment for a project
// @tags project
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param project path int true "Project ID"
// @Param payment body models.ProjectPayment true "The payment."
// @Success 201 {object} models.ProjectPayment
// @Router /projects/{project}/payments [put]
func (pp *ProjectPayment) Create(s *xorm.Session, a web.Auth) error {
	pp.ID = 0
	pp.normalize()
	pp.CreatedByID = a.GetID()
	_, err := s.Insert(pp)
	return err
}

// Update changes a payment.
func (pp *ProjectPayment) Update(s *xorm.Session, _ web.Auth) error {
	pp.normalize()
	_, err := s.Where("id = ? AND project_id = ?", pp.ID, pp.ProjectID).
		Cols("amount", "paid_at", "note").
		Update(pp)
	if err != nil {
		return err
	}
	return pp.ReadOne(s, nil)
}

// Delete removes a payment.
func (pp *ProjectPayment) Delete(s *xorm.Session, _ web.Auth) error {
	_, err := s.Where("id = ? AND project_id = ?", pp.ID, pp.ProjectID).Delete(&ProjectPayment{})
	return err
}

// deleteProjectFinanceData removes the finance details and payments of a
// project. Called when the project itself is deleted.
func deleteProjectFinanceData(s *xorm.Session, projectID int64) error {
	if _, err := s.Where("project_id = ?", projectID).Delete(&ProjectPayment{}); err != nil {
		return err
	}
	_, err := s.Where("project_id = ?", projectID).Delete(&ProjectFinance{})
	return err
}
