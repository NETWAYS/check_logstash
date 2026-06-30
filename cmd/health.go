package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/NETWAYS/check_logstash/internal/logstash"
	"github.com/NETWAYS/go-check"
	"github.com/spf13/cobra"
)

// HealthConfig for the CLI parameters.
type HealthConfig struct {
	FileDescThresWarning  string
	FileDescThresCritical string
	HeapUseThresWarning   string
	HeapUseThresCritical  string
	CPUUseThresWarning    string
	CPUUseThresCritical   string
	UnreachableExitCode   int
}

// HealthThreshold for the parsed CLI parameters.
type HealthThreshold struct {
	fileDescThresWarn *check.Threshold
	fileDescThresCrit *check.Threshold
	heapUseThresWarn  *check.Threshold
	heapUseThresCrit  *check.Threshold
	cpuUseThresWarn   *check.Threshold
	cpuUseThresCrit   *check.Threshold
}

var cliHealthConfig HealthConfig

func parseHealthThresholds(config HealthConfig) (HealthThreshold, error) {
	// Parses the CLI parameters
	var t HealthThreshold
	// File Descriptors
	fileDescThresWarn, err := check.ParseThreshold(config.FileDescThresWarning)
	if err != nil {
		return t, err
	}

	t.fileDescThresWarn = fileDescThresWarn

	fileDescThresCrit, err := check.ParseThreshold(config.FileDescThresCritical)
	if err != nil {
		return t, err
	}

	t.fileDescThresCrit = fileDescThresCrit

	// Heap Usage
	heapUseThresWarn, err := check.ParseThreshold(config.HeapUseThresWarning)
	if err != nil {
		return t, err
	}

	t.heapUseThresWarn = heapUseThresWarn

	heapUseThresCrit, err := check.ParseThreshold(config.HeapUseThresCritical)
	if err != nil {
		return t, err
	}

	t.heapUseThresCrit = heapUseThresCrit

	// CPU Usage
	cpuUseThresWarn, err := check.ParseThreshold(config.CPUUseThresWarning)
	if err != nil {
		return t, err
	}

	t.cpuUseThresWarn = cpuUseThresWarn

	cpuUseThresCrit, err := check.ParseThreshold(config.CPUUseThresCritical)
	if err != nil {
		return t, err
	}

	t.cpuUseThresCrit = cpuUseThresCrit

	return t, nil
}

func generatePerfdata(stat logstash.Stat, thres HealthThreshold) check.PerfdataList {
	// Generates the Perfdata from the results and thresholds
	var l check.PerfdataList

	l.Add(&check.Perfdata{
		Label: "process.cpu.percent",
		Value: stat.Process.CPU.Percent,
		Uom:   "%",
		Warn:  thres.cpuUseThresWarn,
		Crit:  thres.cpuUseThresCrit,
		Min:   0,
		Max:   100})
	l.Add(&check.Perfdata{
		Label: "jvm.mem.heap_used_percent",
		Uom:   "%",
		Value: stat.Jvm.Mem.HeapUsedPercent,
		Warn:  thres.heapUseThresWarn,
		Crit:  thres.heapUseThresCrit,
		Min:   0,
		Max:   100})
	l.Add(&check.Perfdata{
		Label: "jvm.threads.count",
		Value: stat.Jvm.Threads.Count,
		Max:   0})
	l.Add(&check.Perfdata{
		Label: "process.open_file_descriptors",
		Value: stat.Process.OpenFileDescriptors,
		Warn:  thres.fileDescThresWarn,
		Crit:  thres.fileDescThresCrit,
		Min:   0,
		Max:   stat.Process.MaxFileDescriptors})

	return l
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Checks the health of the Logstash server",
	Long:  `Checks the health of the Logstash server`,
	Example: `
	$ check_logstash health --hostname 'localhost' --port 8888 --insecure
	OK - Logstash is healthy | status=green process.cpu.percent=0;0.5;3;0;100
	 \_[OK] Heap usage at 12.00%
	 \_[OK] Open file descriptors at 12.00%
	 \_[OK] CPU usage at 5.00%

	$ check_logstash -p 9600 health --cpu-usage-threshold-warn 50 --cpu-usage-threshold-crit 75
	WARNING - CPU usage at 55.00%
	 \_[OK] Heap usage at 12.00%
	 \_[OK] Open file descriptors at 12.00%
	 \_[WARNING] CPU usage at 55.00%`,
	Run: func(_ *cobra.Command, _ []string) {
		var (
			output     string
			rc         check.Status
			stat       logstash.Stat
			thresholds HealthThreshold
			fdstatus   string
			heapstatus string
			cpustatus  string
		)

		// status + fdstatus + heapstatus + cpustatus = 4
		states := make([]check.Status, 0, 4)

		// Parse the thresholds into a central var since we need them later
		thresholds, err := parseHealthThresholds(cliHealthConfig)
		if err != nil {
			check.ExitError(err)
		}

		unreachableExitCode, errExit := check.NewStatus(cliHealthConfig.UnreachableExitCode)
		if errExit != nil {
			unreachableExitCode = check.Unknown
		}

		// Creating an client and connecting to the API
		c := cliConfig.NewClient()
		u, _ := url.JoinPath(c.URL, "/_node/stats")

		resp, err := c.Client.Get(u)
		if err != nil {
			check.Exit(unreachableExitCode, err.Error())
		}

		if resp.StatusCode != http.StatusOK {
			check.ExitError(fmt.Errorf("could not get %s - Error: %d", u, resp.StatusCode))
		}

		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&stat)
		if err != nil {
			check.ExitError(err)
		}

		// Enable some backwards compatibility
		// Can be changed to a switch statement in the future,
		// when more versions need special cases
		// For Logstash 6, we assume a parsed JSON response
		// is enough to declare the instance running, since there
		// is no status field.
		if stat.MajorVersion == 6 {
			stat.Status = "green"
		}

		// Logstash Health Status
		switch stat.Status {
		default:
			check.ExitError(errors.New("could not determine status"))
		case "green":
			states = append(states, check.OK)
		case "yellow":
			states = append(states, check.Warning)
		case "red":
			states = append(states, check.Critical)
		}

		perfList := generatePerfdata(stat, thresholds)

		// Defaults for the subchecks
		fdstatus = check.OKString
		heapstatus = check.OKString
		cpustatus = check.OKString

		// File Descriptors Check
		fileDescriptorsPercent := (stat.Process.OpenFileDescriptors / stat.Process.MaxFileDescriptors) * 100
		if thresholds.fileDescThresWarn.DoesViolate(fileDescriptorsPercent) {
			states = append(states, check.Warning)
			fdstatus = check.WarningString
		}

		if thresholds.fileDescThresCrit.DoesViolate(fileDescriptorsPercent) {
			states = append(states, check.Critical)
			fdstatus = check.CriticalString
		}

		// Heap Usage Check
		if thresholds.heapUseThresWarn.DoesViolate(stat.Jvm.Mem.HeapUsedPercent) {
			states = append(states, check.Warning)
			heapstatus = check.WarningString
		}

		if thresholds.heapUseThresCrit.DoesViolate(stat.Jvm.Mem.HeapUsedPercent) {
			states = append(states, check.Critical)
			heapstatus = check.CriticalString
		}

		// CPU Usage Check
		if thresholds.cpuUseThresWarn.DoesViolate(stat.Process.CPU.Percent) {
			states = append(states, check.Warning)
			cpustatus = check.WarningString
		}

		if thresholds.cpuUseThresCrit.DoesViolate(stat.Process.CPU.Percent) {
			states = append(states, check.Critical)
			cpustatus = check.CriticalString
		}

		// Validate the various subchecks and use the worst state as return code
		//nolint: exhaustive
		switch check.WorstState(states...) {
		case 0:
			rc = check.OK
			output = "Logstash is healthy"
		case 1:
			rc = check.Warning
			output = "Logstash may not be healthy"
		case 2:
			rc = check.Critical
			output = "Logstash is unhealthy"
		default:
			rc = check.Unknown
			output = "Status unknown"
		}

		// Generate summary for subchecks
		var summary strings.Builder
		fmt.Fprintf(&summary, "\n \\_[%s] Heap usage at %.2f%%", heapstatus, stat.Jvm.Mem.HeapUsedPercent)
		fmt.Fprintf(&summary, "\n \\_[%s] Open file descriptors at %.2f%%", fdstatus, fileDescriptorsPercent)
		fmt.Fprintf(&summary, "\n \\_[%s] CPU usage at %.2f%%", cpustatus, stat.Process.CPU.Percent)

		check.ExitWithPerfdata(rc, perfList, output, summary.String())
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)

	fs := healthCmd.Flags()

	fs.StringVarP(&cliHealthConfig.FileDescThresWarning, "file-descriptor-threshold-warn", "", "100",
		"The percentage relative to the process file descriptor limit on which to be a warning result")
	fs.StringVarP(&cliHealthConfig.FileDescThresCritical, "file-descriptor-threshold-crit", "", "100",
		"The percentage relative to the process file descriptor limit on which to be a critical result")

	fs.StringVarP(&cliHealthConfig.HeapUseThresWarning, "heap-usage-threshold-warn", "", "70",
		"The percentage relative to the heap size limit on which to be a warning result")
	fs.StringVarP(&cliHealthConfig.HeapUseThresCritical, "heap-usage-threshold-crit", "", "80",
		"The percentage relative to the heap size limit on which to be a critical result")

	fs.StringVarP(&cliHealthConfig.CPUUseThresWarning, "cpu-usage-threshold-warn", "", "100",
		"The percentage of CPU usage on which to be a warning result")
	fs.StringVarP(&cliHealthConfig.CPUUseThresCritical, "cpu-usage-threshold-crit", "", "100",
		"The percentage of CPU usage on which to be a critical result")

	fs.IntVarP(&cliHealthConfig.UnreachableExitCode, "unreachable-state", "", 3,
		"Exit with specified code if unreachable. Examples: 1 for Warning, 2 for Critical, 3 for Unknown")

	fs.SortFlags = false
}
