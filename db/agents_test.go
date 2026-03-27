package db

import (
	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent Re-registration with Address Change", func() {
	var (
		db      *DB
		oldAddr = "10.0.0.1:5444"
		newAddr = "10.0.0.2:5444"
	)

	BeforeEach(func() {
		var err error
		db, err = Database()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(db).ShouldNot(BeNil())

		err = db.PreRegisterAgent("10.0.0.1", "test-agent", 5444)
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		db.Disconnect()
	})

	Context("when an agent re-registers with a new IP address", func() {
		It("updates the address in the agents table", func() {
			err := db.PreRegisterAgent("10.0.0.2", "test-agent", 5444)
			Ω(err).ShouldNot(HaveOccurred())

			agents, err := db.GetAllAgents(&AgentFilter{Name: "test-agent"})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(agents).Should(HaveLen(1))
			Ω(agents[0].Address).Should(Equal(newAddr))
		})

		It("makes GetAgentByAddress on the old address return nil", func() {
			err := db.PreRegisterAgent("10.0.0.2", "test-agent", 5444)
			Ω(err).ShouldNot(HaveOccurred())

			agent, err := db.GetAgentByAddress(oldAddr)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(agent).Should(BeNil())
		})

		It("makes GetAgentByAddress on the new address return the agent", func() {
			err := db.PreRegisterAgent("10.0.0.2", "test-agent", 5444)
			Ω(err).ShouldNot(HaveOccurred())

			agent, err := db.GetAgentByAddress(newAddr)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(agent).ShouldNot(BeNil())
			Ω(agent.Name).Should(Equal("test-agent"))
		})

		It("does not create a duplicate agent row", func() {
			err := db.PreRegisterAgent("10.0.0.2", "test-agent", 5444)
			Ω(err).ShouldNot(HaveOccurred())

			agents, err := db.GetAllAgents(&AgentFilter{Name: "test-agent"})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(agents).Should(HaveLen(1))
		})

		It("preserves the agent UUID across address change", func() {
			before, err := db.GetAgentByAddress(oldAddr)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(before).ShouldNot(BeNil())
			originalUUID := before.UUID

			err = db.PreRegisterAgent("10.0.0.2", "test-agent", 5444)
			Ω(err).ShouldNot(HaveOccurred())

			after, err := db.GetAgentByAddress(newAddr)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(after).ShouldNot(BeNil())
			Ω(after.UUID).Should(Equal(originalUUID))
		})
	})

	Context("when a brand-new agent registers", func() {
		It("inserts a fresh row with the given address", func() {
			err := db.PreRegisterAgent("10.0.0.3", "new-agent", 5444)
			Ω(err).ShouldNot(HaveOccurred())

			agent, err := db.GetAgentByAddress("10.0.0.3:5444")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(agent).ShouldNot(BeNil())
			Ω(agent.Name).Should(Equal("new-agent"))
			Ω(agent.Status).Should(Equal("pending"))
		})

		It("leaves existing agents unaffected", func() {
			err := db.PreRegisterAgent("10.0.0.3", "new-agent", 5444)
			Ω(err).ShouldNot(HaveOccurred())

			original, err := db.GetAgentByAddress(oldAddr)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(original).ShouldNot(BeNil())
			Ω(original.Name).Should(Equal("test-agent"))
		})
	})

	Context("when an agent re-registers at the same address", func() {
		It("succeeds without error", func() {
			err := db.PreRegisterAgent("10.0.0.1", "test-agent", 5444)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("does not create a duplicate row", func() {
			err := db.PreRegisterAgent("10.0.0.1", "test-agent", 5444)
			Ω(err).ShouldNot(HaveOccurred())

			agents, err := db.GetAllAgents(&AgentFilter{Name: "test-agent"})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(agents).Should(HaveLen(1))
		})

		It("still returns the agent at the same address", func() {
			err := db.PreRegisterAgent("10.0.0.1", "test-agent", 5444)
			Ω(err).ShouldNot(HaveOccurred())

			agent, err := db.GetAgentByAddress(oldAddr)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(agent).ShouldNot(BeNil())
			Ω(agent.Name).Should(Equal("test-agent"))
		})
	})
})
