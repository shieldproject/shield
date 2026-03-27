package main

import (
	"os"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-table"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/client/v2/shield"
	"github.com/shieldproject/shield/tui"
)

var (
	storesUsed       bool
	storesUnused     bool
	storesWithPlugin string

	storeWithLog    bool
	storeWithoutLog bool

	createStoreName      string
	createStoreSummary   string
	createStoreAgent     string
	createStorePlugin    string
	createStoreThreshold string
	createStoreData      []string

	updateStoreName      string
	updateStoreSummary   string
	updateStoreAgent     string
	updateStorePlugin    string
	updateStoreThreshold string
	updateStoreClearData bool
	updateStoreData      []string

	globalStoresUsed   bool
	globalStoresUnused bool

	globalStoreWithLog    bool
	globalStoreWithoutLog bool

	createGlobalStoreName      string
	createGlobalStoreSummary   string
	createGlobalStoreAgent     string
	createGlobalStorePlugin    string
	createGlobalStoreThreshold string
	createGlobalStoreData      []string

	updateGlobalStoreName      string
	updateGlobalStoreSummary   string
	updateGlobalStoreAgent     string
	updateGlobalStorePlugin    string
	updateGlobalStoreThreshold string
	updateGlobalStoreClearData bool
	updateGlobalStoreData      []string
)

var storesCmd = &cobra.Command{
	Use:   "stores [NAME-or-UUID]",
	Short: "List backup stores in the current tenant",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		required(!(storesUsed && storesUnused), "The --used and --unused options are mutually exclusive.")

		c := clientFromConfig()
		filter := &shield.StoreFilter{
			Plugin: storesWithPlugin,
			Fuzzy:  !optExact,
		}
		if len(args) == 1 {
			filter.Name = args[0]
			filter.UUID = args[0]
		}
		if storesUsed || storesUnused {
			x := storesUsed
			filter.Used = &x
		}

		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		stores, err := c.ListStores(tenant, filter)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(stores))
			return nil
		}

		tbl := table.NewTable("UUID", "Name", "Summary", "Plugin", "SHIELD Agent", "Healthy?")
		for _, store := range stores {
			health := fmt.Sprintf("@G{yes}")
			if !store.Healthy {
				health = fmt.Sprintf("@R{no}")
			}
			tbl.Row(store, uuid8full(store.UUID, optLong), store.Name, wrap(store.Summary, 35), store.Plugin, store.Agent, health)
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var storeCmd = &cobra.Command{
	Use:   "store NAME-or-UUID",
	Short: "Show details of a backup store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		required(!(storeWithLog && storeWithoutLog), "The --with-log and --without-log options are mutually exclusive.")
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		store, err := c.FindStore(tenant, args[0], !optExact)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(store))
			return nil
		}

		health := fmt.Sprintf("@G{yes}")
		if !store.Healthy {
			health = fmt.Sprintf("@R{no}")
			if !storeWithoutLog {
				storeWithLog = true
			}
		}

		var tasks []*shield.Task
		if storeWithLog && !storeWithoutLog {
			limit := 1
			active := false
			tasks, err = c.ListTasks(tenant, &shield.TaskFilter{
				Store:  store.UUID,
				Type:   "test-store",
				Limit:  &limit,
				Active: &active,
			})
			bail(err)
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Healthy?", health)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Storage Plugin", store.Plugin)
		r.Break()
		r.Add("Configuration", asJSON(store.Config))
		if len(tasks) == 1 {
			r.Break()
			r.Add("Last Checked", strftime(tasks[0].RequestedAt))
			r.Add("Check Task", tasks[0].UUID)
			r.Add("Status", tasks[0].Status)
			r.Break()
			r.Add("Task Log", tasks[0].Log)
		} else if storeWithLog {
			r.Break()
			r.Add("Last Checked", "(unknown)")
			r.Add("Check Task", "(unknown)")
			r.Add("Status", "(unknown)")
		}
		r.Output(os.Stdout)
		return nil
	},
}

var createStoreCmd = &cobra.Command{
	Use:   "create-store",
	Short: "Create a new backup store in the current tenant",
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		conf, err := dataConfig(createStoreData)
		bail(err)

		if !optBatch {
			if createStoreName == "" {
				createStoreName = prompt("@C{Store Name}: ")
			}
			if createStoreSummary == "" {
				createStoreSummary = prompt("@C{Description}: ")
			}
			if createStoreAgent == "" {
				createStoreAgent = prompt("@C{SHIELD Agent (IP:port)}: ")
			}
			if createStorePlugin == "" {
				createStorePlugin = prompt("@C{Backup Plugin}: ")
			}
			if createStoreThreshold == "" {
				createStoreThreshold = prompt("@C{Threshold}: ")
			}
		}

		thold, err := parseBytes(createStoreThreshold)
		bail(err)

		store, err := c.CreateStore(tenant, &shield.Store{
			Name:      createStoreName,
			Summary:   createStoreSummary,
			Agent:     createStoreAgent,
			Plugin:    createStorePlugin,
			Threshold: thold,
			Config:    conf,
		})
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(store))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		r.Add("Threshold", formatBytes(store.Threshold))
		r.Output(os.Stdout)
		return nil
	},
}

var updateStoreCmd = &cobra.Command{
	Use:   "update-store NAME-or-UUID",
	Short: "Update an existing backup store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		store, err := c.FindStore(tenant, args[0], true)
		bail(err)

		conf, err := dataConfig(updateStoreData)
		bail(err)

		if updateStoreName != "" {
			store.Name = updateStoreName
		}
		if updateStoreSummary != "" {
			store.Summary = updateStoreSummary
		}
		if updateStoreAgent != "" {
			store.Agent = updateStoreAgent
		}
		if updateStorePlugin != "" && store.Plugin != updateStorePlugin {
			updateStoreClearData = true
			store.Plugin = updateStorePlugin
		}
		if updateStoreThreshold != "" {
			thold, err := parseBytes(updateStoreThreshold)
			fmt.Printf("threshold is '%s' -> %d\n", updateStoreThreshold, thold)
			bail(err)
			store.Threshold = thold
		}
		if store.Config == nil {
			store.Config = make(map[string]interface{})
		}
		if updateStoreClearData {
			store.Config = conf
		} else {
			for k, v := range conf {
				store.Config[k] = v
			}
		}

		_, err = c.UpdateStore(tenant, store)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(store))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		r.Add("Threshold", formatBytes(store.Threshold))
		r.Output(os.Stdout)
		return nil
	},
}

var deleteStoreCmd = &cobra.Command{
	Use:   "delete-store NAME-or-UUID",
	Short: "Delete a backup store from the current tenant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		store, err := c.FindStore(tenant, args[0], true)
		bail(err)

		if !confirm(optYes, "Delete store @Y{%s} in tenant @Y{%s}?", store.Name, tenant.Name) {
			return nil
		}
		r, err := c.DeleteStore(tenant, store)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(r))
			return nil
		}
		fmt.Printf("%s\n", r.OK)
		return nil
	},
}

var globalStoresCmd = &cobra.Command{
	Use:   "global-stores [NAME-or-UUID]",
	Short: "List global backup stores",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		required(!(globalStoresUsed && globalStoresUnused), "The --used and --unused options are mutually exclusive.")

		c := clientFromConfig()
		filter := &shield.StoreFilter{
			Plugin: storesWithPlugin,
			Fuzzy:  !optExact,
		}
		if len(args) == 1 {
			filter.Name = args[0]
			filter.UUID = args[0]
		}
		if globalStoresUsed || globalStoresUnused {
			x := globalStoresUsed
			filter.Used = &x
		}

		stores, err := c.ListGlobalStores(filter)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(stores))
			return nil
		}

		tbl := table.NewTable("UUID", "Name", "Summary", "Plugin", "SHIELD Agent", "Healthy?")
		for _, store := range stores {
			health := fmt.Sprintf("@G{yes}")
			if !store.Healthy {
				health = fmt.Sprintf("@R{no}")
			}
			tbl.Row(store, uuid8full(store.UUID, optLong), store.Name, wrap(store.Summary, 35), store.Plugin, store.Agent, health)
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var globalStoreCmd = &cobra.Command{
	Use:   "global-store NAME-or-UUID",
	Short: "Show details of a global backup store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		required(!(globalStoreWithLog && globalStoreWithoutLog), "The --with-log and --without-log options are mutually exclusive.")

		c := clientFromConfig()
		store, err := c.FindGlobalStore(args[0], !optExact)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(store))
			return nil
		}

		health := fmt.Sprintf("@G{yes}")
		if !store.Healthy {
			health = fmt.Sprintf("@R{no}")
			if !globalStoreWithoutLog {
				globalStoreWithLog = true
			}
		}

		var tasks []*shield.Task
		if globalStoreWithLog && !globalStoreWithoutLog {
			limit := 1
			active := false
			tasks, err = c.ListTasks(nil, &shield.TaskFilter{
				Store:  store.UUID,
				Type:   "test-store",
				Limit:  &limit,
				Active: &active,
			})
			bail(err)
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Healthy?", health)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		if len(tasks) == 1 {
			r.Break()
			r.Add("Last Checked", strftime(tasks[0].RequestedAt))
			r.Add("Check Task", tasks[0].UUID)
			r.Add("Status", tasks[0].Status)
			r.Break()
			r.Add("Task Log", tasks[0].Log)
		} else if globalStoreWithLog {
			r.Break()
			r.Add("Last Checked", "(unknown)")
			r.Add("Check Task", "(unknown)")
			r.Add("Status", "(unknown)")
		}
		r.Output(os.Stdout)
		return nil
	},
}

var createGlobalStoreCmd = &cobra.Command{
	Use:   "create-global-store",
	Short: "Create a new global backup store",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		conf, err := dataConfig(createGlobalStoreData)
		bail(err)

		if !optBatch {
			if createGlobalStoreName == "" {
				createGlobalStoreName = prompt("@C{Store Name}: ")
			}
			if createGlobalStoreSummary == "" {
				createGlobalStoreSummary = prompt("@C{Description}: ")
			}
			if createGlobalStoreAgent == "" {
				createGlobalStoreAgent = prompt("@C{SHIELD Agent (IP:port)}: ")
			}
			if createGlobalStorePlugin == "" {
				createGlobalStorePlugin = prompt("@C{Backup Plugin}: ")
			}
			if createGlobalStoreThreshold == "" {
				createGlobalStoreThreshold = prompt("@C{Threshold}: ")
			}
		}

		thold, err := parseBytes(createGlobalStoreThreshold)
		bail(err)

		store, err := c.CreateGlobalStore(&shield.Store{
			Name:      createGlobalStoreName,
			Summary:   createGlobalStoreSummary,
			Agent:     createGlobalStoreAgent,
			Plugin:    createGlobalStorePlugin,
			Threshold: thold,
			Config:    conf,
		})
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(store))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		r.Add("Threshold", formatBytes(store.Threshold))
		r.Output(os.Stdout)
		return nil
	},
}

var updateGlobalStoreCmd = &cobra.Command{
	Use:   "update-global-store NAME-or-UUID",
	Short: "Update an existing global backup store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		store, err := c.FindGlobalStore(args[0], true)
		bail(err)

		conf, err := dataConfig(updateGlobalStoreData)
		bail(err)

		if updateGlobalStoreName != "" {
			store.Name = updateGlobalStoreName
		}
		if updateGlobalStoreSummary != "" {
			store.Summary = updateGlobalStoreSummary
		}
		if updateGlobalStoreAgent != "" {
			store.Agent = updateGlobalStoreAgent
		}
		if updateGlobalStorePlugin != "" && store.Plugin != updateGlobalStorePlugin {
			updateGlobalStoreClearData = true
			store.Plugin = updateGlobalStorePlugin
		}
		if updateGlobalStoreThreshold != "" {
			thold, err := parseBytes(updateGlobalStoreThreshold)
			bail(err)
			store.Threshold = thold
		}
		if store.Config == nil {
			store.Config = make(map[string]interface{})
		}
		if updateGlobalStoreClearData {
			store.Config = conf
		} else {
			for k, v := range conf {
				store.Config[k] = v
			}
		}

		_, err = c.UpdateGlobalStore(store)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(store))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		r.Add("Threshold", formatBytes(store.Threshold))
		r.Output(os.Stdout)
		return nil
	},
}

var deleteGlobalStoreCmd = &cobra.Command{
	Use:   "delete-global-store NAME-or-UUID",
	Short: "Delete a global backup store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		store, err := c.FindGlobalStore(args[0], true)
		bail(err)

		if !confirm(optYes, "Delete @R{global} store @Y{%s}?", store.Name) {
			return nil
		}
		r, err := c.DeleteGlobalStore(store)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(r))
			return nil
		}
		fmt.Printf("%s\n", r.OK)
		return nil
	},
}

func init() {
	storesCmd.Flags().BoolVar(&storesUsed, "used", false, "Show only stores that are in use")
	storesCmd.Flags().BoolVar(&storesUnused, "unused", false, "Show only stores not in use")
	storesCmd.Flags().StringVar(&storesWithPlugin, "with-plugin", "", "Filter by plugin name")

	storeCmd.Flags().BoolVar(&storeWithLog, "with-log", false, "Show the last test-store task log")
	storeCmd.Flags().BoolVar(&storeWithoutLog, "without-log", false, "Never show the task log")

	createStoreCmd.Flags().StringVarP(&createStoreName, "name", "n", "", "Store name")
	createStoreCmd.Flags().StringVarP(&createStoreSummary, "summary", "s", "", "Store description")
	createStoreCmd.Flags().StringVarP(&createStoreAgent, "agent", "a", "", "SHIELD agent address (IP:port)")
	createStoreCmd.Flags().StringVarP(&createStorePlugin, "plugin", "p", "", "Storage plugin name")
	createStoreCmd.Flags().StringVar(&createStoreThreshold, "threshold", "", "Storage threshold (e.g. 100M)")
	createStoreCmd.Flags().StringArrayVarP(&createStoreData, "data", "d", nil, "Plugin configuration (key=value)")

	updateStoreCmd.Flags().StringVarP(&updateStoreName, "name", "n", "", "New store name")
	updateStoreCmd.Flags().StringVarP(&updateStoreSummary, "summary", "s", "", "New store description")
	updateStoreCmd.Flags().StringVarP(&updateStoreAgent, "agent", "a", "", "New SHIELD agent address (IP:port)")
	updateStoreCmd.Flags().StringVarP(&updateStorePlugin, "plugin", "p", "", "New storage plugin name")
	updateStoreCmd.Flags().StringVar(&updateStoreThreshold, "threshold", "", "New storage threshold")
	updateStoreCmd.Flags().BoolVar(&updateStoreClearData, "clear-data", false, "Clear existing plugin configuration")
	updateStoreCmd.Flags().StringArrayVarP(&updateStoreData, "data", "d", nil, "Plugin configuration (key=value)")

	globalStoresCmd.Flags().BoolVar(&globalStoresUsed, "used", false, "Show only stores that are in use")
	globalStoresCmd.Flags().BoolVar(&globalStoresUnused, "unused", false, "Show only stores not in use")

	globalStoreCmd.Flags().BoolVar(&globalStoreWithLog, "with-log", false, "Show the last test-store task log")
	globalStoreCmd.Flags().BoolVar(&globalStoreWithoutLog, "without-log", false, "Never show the task log")

	createGlobalStoreCmd.Flags().StringVarP(&createGlobalStoreName, "name", "n", "", "Store name")
	createGlobalStoreCmd.Flags().StringVarP(&createGlobalStoreSummary, "summary", "s", "", "Store description")
	createGlobalStoreCmd.Flags().StringVarP(&createGlobalStoreAgent, "agent", "a", "", "SHIELD agent address (IP:port)")
	createGlobalStoreCmd.Flags().StringVarP(&createGlobalStorePlugin, "plugin", "p", "", "Storage plugin name")
	createGlobalStoreCmd.Flags().StringVar(&createGlobalStoreThreshold, "threshold", "", "Storage threshold (e.g. 100M)")
	createGlobalStoreCmd.Flags().StringArrayVarP(&createGlobalStoreData, "data", "d", nil, "Plugin configuration (key=value)")

	updateGlobalStoreCmd.Flags().StringVarP(&updateGlobalStoreName, "name", "n", "", "New store name")
	updateGlobalStoreCmd.Flags().StringVarP(&updateGlobalStoreSummary, "summary", "s", "", "New store description")
	updateGlobalStoreCmd.Flags().StringVarP(&updateGlobalStoreAgent, "agent", "a", "", "New SHIELD agent address (IP:port)")
	updateGlobalStoreCmd.Flags().StringVarP(&updateGlobalStorePlugin, "plugin", "p", "", "New storage plugin name")
	updateGlobalStoreCmd.Flags().StringVar(&updateGlobalStoreThreshold, "threshold", "", "New storage threshold")
	updateGlobalStoreCmd.Flags().BoolVar(&updateGlobalStoreClearData, "clear-data", false, "Clear existing plugin configuration")
	updateGlobalStoreCmd.Flags().StringArrayVarP(&updateGlobalStoreData, "data", "d", nil, "Plugin configuration (key=value)")

	rootCmd.AddCommand(storesCmd)
	rootCmd.AddCommand(storeCmd)
	rootCmd.AddCommand(createStoreCmd)
	rootCmd.AddCommand(updateStoreCmd)
	rootCmd.AddCommand(deleteStoreCmd)
	rootCmd.AddCommand(globalStoresCmd)
	rootCmd.AddCommand(globalStoreCmd)
	rootCmd.AddCommand(createGlobalStoreCmd)
	rootCmd.AddCommand(updateGlobalStoreCmd)
	rootCmd.AddCommand(deleteGlobalStoreCmd)
}
