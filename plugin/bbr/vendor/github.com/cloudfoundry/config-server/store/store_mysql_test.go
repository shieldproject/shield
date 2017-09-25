package store_test

import (
	. "github.com/cloudfoundry/config-server/store"

	"database/sql"
	"errors"
	fakes "github.com/cloudfoundry/config-server/store/storefakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StoreMysql", func() {

	var (
		fakeDbProvider *fakes.FakeDbProvider
		fakeDb         *fakes.FakeIDb
		fakeRow        *fakes.FakeIRow
		fakeRows       *fakes.FakeIRows
		fakeResult     *fakes.FakeResult

		store Store
	)

	BeforeEach(func() {
		fakeDbProvider = &fakes.FakeDbProvider{}
		fakeDb = &fakes.FakeIDb{}
		fakeRow = &fakes.FakeIRow{}
		fakeRows = &fakes.FakeIRows{}
		fakeResult = &fakes.FakeResult{}

		store = NewMysqlStore(fakeDbProvider)
	})

	Describe("GetByName", func() {
		It("queries the database for the latest entry for a given name", func() {
			fakeDb.QueryReturns(fakeRows, nil)
			fakeDbProvider.DbReturns(fakeDb, nil)

			_, err := store.GetByName("Luke")
			Expect(err).To(BeNil())
			query, _ := fakeDb.QueryArgsForCall(0)

			Expect(query).To(Equal("SELECT id, name, value FROM configurations WHERE name = ? ORDER BY id DESC"))
		})

		It("returns ALL values from db query", func() {
			var rawConfigs = []Configuration{
				{
					ID:    "6",
					Name:  "someName",
					Value: "someOtherValue",
				},
				{
					ID:    "5",
					Name:  "someName",
					Value: "someValue",
				},
			}
			var index int = -1

			fakeRows.NextStub = func() bool {
				index++
				return index < len(rawConfigs)
			}

			fakeRows.ScanStub = func(dest ...interface{}) error {
				idPtr, ok := dest[0].(*string)
				Expect(ok).To(BeTrue())

				*idPtr = rawConfigs[index].ID
				namePtr, ok := dest[1].(*string)
				Expect(ok).To(BeTrue())

				*namePtr = rawConfigs[index].Name
				valuePtr, ok := dest[2].(*string)

				Expect(ok).To(BeTrue())
				*valuePtr = rawConfigs[index].Value

				return nil
			}

			fakeDb.QueryReturns(fakeRows, nil)
			fakeDbProvider.DbReturns(fakeDb, nil)

			values, err := store.GetByName("someName")
			Expect(err).To(BeNil())
			Expect(values[0]).To(Equal(rawConfigs[0]))
			Expect(values[1]).To(Equal(rawConfigs[1]))
		})

		It("returns empty configuration array when no result is found", func() {
			fakeRow.ScanReturns(sql.ErrNoRows)

			fakeDb.QueryReturns(fakeRows, nil)
			fakeDbProvider.DbReturns(fakeDb, nil)

			values, err := store.GetByName("luke")
			Expect(err).To(BeNil())
			Expect(len(values)).To(Equal(0))
		})

		It("returns an error when db provider fails to return db", func() {
			dbError := errors.New("connection failure")
			fakeDbProvider.DbReturns(nil, dbError)

			_, err := store.GetByName("luke")
			Expect(err).ToNot(BeNil())
			Expect(err).To(Equal(dbError))
		})

		It("returns an error when db query fails", func() {
			queryError := errors.New("query failure")

			fakeDb.QueryReturns(fakeRows, queryError)
			fakeDbProvider.DbReturns(fakeDb, nil)

			_, err := store.GetByName("luke")
			Expect(err).ToNot(BeNil())
			Expect(err).To(Equal(queryError))
		})
	})

	Describe("GetById", func() {
		It("queries the database for the latest entry for a given id", func() {
			fakeDb.QueryRowReturns(&fakes.FakeIRow{})
			fakeDbProvider.DbReturns(fakeDb, nil)

			_, err := store.GetByID("1")
			Expect(err).To(BeNil())
			query, _ := fakeDb.QueryRowArgsForCall(0)

			Expect(query).To(Equal("SELECT id, name, value FROM configurations WHERE id = ?"))
		})

		It("returns value from db query", func() {
			fakeRow.ScanStub = func(dest ...interface{}) error {
				idPtr, ok := dest[0].(*string)
				Expect(ok).To(BeTrue())

				namePtr, ok := dest[1].(*string)
				Expect(ok).To(BeTrue())

				valuePtr, ok := dest[2].(*string)
				Expect(ok).To(BeTrue())

				*idPtr = "54"
				*valuePtr = "Skywalker"
				*namePtr = "Luke"

				return nil
			}

			fakeDb.QueryRowReturns(fakeRow)
			fakeDbProvider.DbReturns(fakeDb, nil)

			value, err := store.GetByID("54")
			Expect(err).To(BeNil())
			Expect(value).To(Equal(Configuration{
				ID:    "54",
				Value: "Skywalker",
				Name:  "Luke",
			}))
		})

		It("returns empty configuration when no result is found", func() {
			fakeRow.ScanReturns(sql.ErrNoRows)

			fakeDb.QueryRowReturns(fakeRow)
			fakeDbProvider.DbReturns(fakeDb, nil)

			value, err := store.GetByID("54")
			Expect(err).To(BeNil())
			Expect(value).To(Equal(Configuration{}))
		})

		It("returns an error when db provider fails to return db", func() {
			dbError := errors.New("connection failure")
			fakeDbProvider.DbReturns(nil, dbError)

			_, err := store.GetByID("2")
			Expect(err).ToNot(BeNil())
			Expect(err).To(Equal(dbError))
		})

		It("returns an error when db query fails", func() {
			scanError := errors.New("query failure")
			fakeRow.ScanReturns(scanError)

			fakeDb.QueryRowReturns(fakeRow)
			fakeDbProvider.DbReturns(fakeDb, nil)

			_, err := store.GetByID("7")
			Expect(err).ToNot(BeNil())
			Expect(err).To(Equal(scanError))
		})
	})

	Describe("Put", func() {
		It("does an insert to the database", func() {
			fakeDbProvider.DbReturns(fakeDb, nil)
			fakeDb.ExecReturns(fakeResult, nil)

			_, err := store.Put("Luke", "Skywalker")
			Expect(err).To(BeNil())

			Expect(fakeDb.ExecCallCount()).To(Equal(1))

			query, values := fakeDb.ExecArgsForCall(0)
			Expect(query).To(Equal("INSERT INTO configurations (name, value) VALUES(?,?)"))

			Expect(values[0]).To(Equal("Luke"))
			Expect(values[1]).To(Equal("Skywalker"))
		})

		It("returns id of new record", func() {
			fakeDbProvider.DbReturns(fakeDb, nil)
			fakeDb.ExecReturns(fakeResult, nil)
			fakeResult.LastInsertIdReturns(9, nil)

			id, err := store.Put("Luke", "Skywalker")
			Expect(err).To(BeNil())
			Expect(id).To(Equal("9"))
		})
	})

	Describe("Delete", func() {
		Context("Name exists", func() {

			BeforeEach(func() {
				fakeDbProvider.DbReturns(fakeDb, nil)
				fakeDb.ExecReturns(fakeResult, nil)

				fakeResult.RowsAffectedReturns(1, nil)
			})

			It("removes value", func() {
				store.Delete("Luke")

				Expect(fakeDb.ExecCallCount()).To(Equal(1))
				query, value := fakeDb.ExecArgsForCall(0)
				Expect(query).To(Equal("DELETE FROM configurations WHERE name = ?"))
				Expect(value[0]).To(Equal("Luke"))
			})

			It("returns count of deleted rows", func() {
				deleted, err := store.Delete("Luke")

				Expect(deleted).To(Equal(1))
				Expect(err).To(BeNil())
			})
		})

		Context("Name does not exist", func() {

			BeforeEach(func() {
				fakeDbProvider.DbReturns(fakeDb, nil)
				fakeDb.ExecReturns(fakeResult, nil)

				fakeResult.RowsAffectedReturns(0, nil)
			})

			It("returns count of deleted rows", func() {
				deleted, err := store.Delete("name")
				Expect(deleted).To(Equal(0))
				Expect(err).To(BeNil())
			})
		})
	})
})
