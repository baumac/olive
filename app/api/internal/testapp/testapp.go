// Package testapp gives the tests one migrated database for each test
// binary. See decision 00045.
package testapp

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"golang.org/x/crypto/bcrypt"
)

type template struct {
	once sync.Once
	dir  string
	err  error
}

var (
	migrated template
	lowCost  template
)

// Dir returns a data folder with all the migrations that the test binary
// registers. tests.NewTestApp copies it for each app.
func Dir(t testing.TB) string {
	t.Helper()
	return migrated.get(t, func() (string, error) {
		empty, err := os.MkdirTemp("", "reviews_template_*")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(empty)
		return makeTemplate(empty, func(core.App) error { return nil })
	})
}

// LowCostDir returns a copy of Dir with the bcrypt cost bcrypt.MinCost in
// each password field. Production has the default cost.
func LowCostDir(t testing.TB) string {
	t.Helper()
	dir := Dir(t)
	return lowCost.get(t, func() (string, error) {
		return makeTemplate(dir, lowerPasswordCost)
	})
}

// Main runs the tests of a package. It then deletes the data folders of Dir
// and LowCostDir.
func Main(m *testing.M) {
	code := m.Run()
	for _, dir := range []string{lowCost.dir, migrated.dir} {
		if dir == "" {
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			code = 1
		}
	}
	os.Exit(code)
}

func (tp *template) get(t testing.TB, build func() (string, error)) string {
	t.Helper()
	tp.once.Do(func() { tp.dir, tp.err = build() })
	if tp.err != nil {
		t.Fatal(tp.err)
	}
	return tp.dir
}

// makeTemplate runs change on a migrated copy of from.
func makeTemplate(from string, change func(core.App) error) (string, error) {
	app, err := tests.NewTestApp(from)
	if err != nil {
		return "", err
	}
	if err := change(app); err != nil {
		app.Cleanup()
		return "", err
	}
	// ResetBootstrapState closes the databases. The data folder stays.
	if err := app.ResetBootstrapState(); err != nil {
		app.Cleanup()
		return "", err
	}
	return app.DataDir(), nil
}

// lowerPasswordCost makes each bcrypt hash about 64 times faster than the
// default cost. The race detector makes one hash at the default cost take
// about half a second.
func lowerPasswordCost(app core.App) error {
	collections, err := app.FindAllCollections(core.CollectionTypeAuth)
	if err != nil {
		return err
	}
	if len(collections) == 0 {
		return errors.New("testapp: no auth collection")
	}
	for _, c := range collections {
		f, ok := c.Fields.GetByName(core.FieldNamePassword).(*core.PasswordField)
		if !ok {
			return errors.New("testapp: no password field in " + c.Name)
		}
		f.Cost = bcrypt.MinCost
		if err := app.Save(c); err != nil {
			return err
		}
	}
	return nil
}
