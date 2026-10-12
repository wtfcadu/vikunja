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
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectFinance_Permissions(t *testing.T) {
	owner := &user.User{ID: 1}
	// user 1 has admin permission on project 11 but does not own it.
	t.Run("owner can read", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		can, perm, err := (&ProjectFinance{ProjectID: 1}).CanRead(s, owner)
		require.NoError(t, err)
		assert.True(t, can)
		assert.Equal(t, int(PermissionAdmin), perm)
	})
	t.Run("shared admin who is not the owner cannot read", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		can, _, err := (&ProjectFinance{ProjectID: 11}).CanRead(s, owner)
		require.NoError(t, err)
		assert.False(t, can)
	})
	t.Run("shared admin who is not the owner cannot update", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		can, err := (&ProjectFinance{ProjectID: 11}).CanUpdate(s, owner)
		require.NoError(t, err)
		assert.False(t, can)
	})
	t.Run("link share cannot read", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		can, _, err := (&ProjectFinance{ProjectID: 1}).CanRead(s, &LinkSharing{ID: 1, ProjectID: 1, Permission: PermissionAdmin})
		require.NoError(t, err)
		assert.False(t, can)
	})
	t.Run("nonexistent project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		_, _, err := (&ProjectFinance{ProjectID: 99999}).CanRead(s, owner)
		require.Error(t, err)
		assert.True(t, IsErrProjectDoesNotExist(err))
	})
}

func TestProjectFinance_ReadAndUpdate(t *testing.T) {
	u := &user.User{ID: 1}
	t.Run("read existing", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		pf := &ProjectFinance{ProjectID: 1}
		require.NoError(t, pf.ReadOne(s, u))
		assert.Equal(t, "Acme Studio", pf.Client)
		assert.Equal(t, "GBP", pf.Currency)
		assert.Equal(t, int64(100000), pf.Fee)
	})
	t.Run("read missing returns empty details", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		pf := &ProjectFinance{ProjectID: 6}
		require.NoError(t, pf.ReadOne(s, u))
		assert.Equal(t, int64(6), pf.ProjectID)
		assert.Empty(t, pf.Client)
		assert.Zero(t, pf.Fee)
	})
	t.Run("update creates details on first save", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		pf := &ProjectFinance{ProjectID: 6, Client: " New Client ", Currency: "usd", Fee: 2500}
		require.NoError(t, pf.Update(s, u))
		require.NoError(t, s.Commit())
		assert.Equal(t, "New Client", pf.Client)
		assert.Equal(t, "USD", pf.Currency)
		db.AssertExists(t, "project_finances", map[string]interface{}{
			"project_id": 6,
			"client":     "New Client",
			"currency":   "USD",
			"fee":        2500,
		}, false)
	})
	t.Run("update existing details", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		pf := &ProjectFinance{ProjectID: 1, Client: "Renamed", Currency: "EUR", Fee: 9000}
		require.NoError(t, pf.Update(s, u))
		require.NoError(t, s.Commit())
		db.AssertExists(t, "project_finances", map[string]interface{}{
			"project_id": 1,
			"client":     "Renamed",
			"currency":   "EUR",
			"fee":        9000,
		}, false)
	})
	t.Run("invalid currency", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		err := (&ProjectFinance{ProjectID: 1, Currency: "POUNDS"}).Update(s, u)
		require.Error(t, err)
		assert.True(t, IsErrInvalidCurrency(err))
	})
}

func TestProjectPayment(t *testing.T) {
	owner := &user.User{ID: 1}
	t.Run("list payments newest first", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		result, count, total, err := (&ProjectPayment{ProjectID: 1}).ReadAll(s, owner, "", 0, 0)
		require.NoError(t, err)
		payments, ok := result.([]*ProjectPayment)
		require.True(t, ok)
		assert.Equal(t, 2, count)
		assert.Equal(t, int64(2), total)
		assert.Equal(t, int64(2), payments[0].ID)
		assert.Equal(t, int64(1), payments[1].ID)
	})
	t.Run("owner can create", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		pp := &ProjectPayment{ProjectID: 1, Amount: 5000, Note: "Final", PaidAt: time.Now()}
		can, err := pp.CanCreate(s, owner)
		require.NoError(t, err)
		require.True(t, can)
		require.NoError(t, pp.Create(s, owner))
		require.NoError(t, s.Commit())
		db.AssertExists(t, "project_payments", map[string]interface{}{
			"id":            pp.ID,
			"project_id":    1,
			"amount":        5000,
			"note":          "Final",
			"created_by_id": 1,
		}, false)
	})
	t.Run("non-owner cannot create", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		can, err := (&ProjectPayment{ProjectID: 11, Amount: 1}).CanCreate(s, owner)
		require.NoError(t, err)
		assert.False(t, can)
	})
	t.Run("payment from another project is not reachable", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		// Payment 3 belongs to project 11, not project 1.
		_, err := (&ProjectPayment{ID: 3, ProjectID: 1}).CanUpdate(s, owner)
		require.Error(t, err)
		assert.True(t, IsErrProjectPaymentDoesNotExist(err))
	})
	t.Run("owner can update", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		pp := &ProjectPayment{ID: 1, ProjectID: 1, Amount: 35000, Note: "Bigger deposit", PaidAt: time.Now()}
		can, err := pp.CanUpdate(s, owner)
		require.NoError(t, err)
		require.True(t, can)
		require.NoError(t, pp.Update(s, owner))
		require.NoError(t, s.Commit())
		db.AssertExists(t, "project_payments", map[string]interface{}{
			"id":     1,
			"amount": 35000,
			"note":   "Bigger deposit",
		}, false)
	})
	t.Run("owner can delete", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		pp := &ProjectPayment{ID: 1, ProjectID: 1}
		can, err := pp.CanDelete(s, owner)
		require.NoError(t, err)
		require.True(t, can)
		require.NoError(t, pp.Delete(s, owner))
		require.NoError(t, s.Commit())
		db.AssertMissing(t, "project_payments", map[string]interface{}{"id": 1})
	})
	t.Run("non-owner cannot delete", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		can, err := (&ProjectPayment{ID: 3, ProjectID: 11}).CanDelete(s, owner)
		require.NoError(t, err)
		assert.False(t, can)
	})
}

func TestGetProjectOverview(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	items, err := GetProjectOverview(s, &user.User{ID: 1})
	require.NoError(t, err)
	require.NotEmpty(t, items)

	var p1 *ProjectOverviewItem
	for _, item := range items {
		// Only owned projects are listed, so project 11 (owner 6) must be absent.
		assert.NotEqual(t, int64(11), item.ID)
		if item.ID == 1 {
			p1 = item
		}
	}
	require.NotNil(t, p1)
	assert.Equal(t, "Acme Studio", p1.Client)
	assert.Equal(t, "GBP", p1.Currency)
	assert.Equal(t, int64(100000), p1.Fee)
	assert.Equal(t, int64(50000), p1.Paid)
	assert.Equal(t, int64(50000), p1.Owed)
	assert.Equal(t, int64(2), p1.Payments)
	assert.Positive(t, p1.OpenTasks+p1.DoneTasks)
}

func TestProjectDelete_RemovesFinanceData(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	p := &Project{ID: 1}
	require.NoError(t, p.Delete(s, &user.User{ID: 1}))
	require.NoError(t, s.Commit())
	db.AssertMissing(t, "project_finances", map[string]interface{}{"project_id": 1})
	db.AssertMissing(t, "project_payments", map[string]interface{}{"project_id": 1})
}
