package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/NETWAYS/check_logstash/internal/logstash"
	"github.com/NETWAYS/go-check"
	"github.com/NETWAYS/go-check/perfdata"
	"github.com/NETWAYS/go-check/result"
	"github.com/spf13/cobra"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// To store the CLI parameters
type PipelineConfig struct {
	PipelineName string
	Warning      string
	Critical     string
}

// To store the parsed CLI parameters
type PipelineThreshold struct {
	Warning  *check.Threshold
	Critical *check.Threshold
}

var cliPipelineConfig PipelineConfig

func parsePipeThresholds(config PipelineConfig) (PipelineThreshold, error) {
	// Parses the CLI parameters
	var t PipelineThreshold

	warn, err := check.ParseThreshold(config.Warning)
	if err != nil {
		return t, err
	}

	t.Warning = warn

	crit, err := check.ParseThreshold(config.Critical)
	if err != nil {
		return t, err
	}

	t.Critical = crit

	return t, nil
}

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Checks the status of the Logstash Pipelines",
	Long:  `Checks the status of the Logstash Pipelines`,
	Example: `
	$ check_logstash pipeline --inflight-events-warn 5 --inflight-events-crit 10
	WARNING - Inflight events
	 \_[WARNING] inflight_events_example-input:9;
	 \_[OK] inflight_events_example-default-connector:4

	$ check_logstash pipeline --inflight-events-warn 5 --inflight-events-crit 10 --pipeline example
	CRITICAL - Inflight events
	 \_[CRITICAL] inflight_events_example:15`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			output     string
			rc         int
			pp         logstash.Pipeline
			thresholds PipelineThreshold
			perfList   perfdata.PerfdataList
		)

		// Parse the thresholds into a central var since we need them later
		thresholds, err := parsePipeThresholds(cliPipelineConfig)
		if err != nil {
			check.ExitError(err)
		}

		// Creating an client and connecting to the API
		c := cliConfig.NewClient()
		// localhost:9600/_node/stats/pipelines/ will return all Pipelines
		// localhost:9600/_node/stats/pipelines/foo will return the foo Pipeline
		u, _ := url.JoinPath(c.Url, "/_node/stats/pipelines", cliPipelineConfig.PipelineName)
		resp, err := c.Client.Get(u)

		if err != nil {
			check.ExitError(err)
		}

		if resp.StatusCode != http.StatusOK {
			check.ExitError(fmt.Errorf("Could not get %s - Error: %d", u, resp.StatusCode))
		}

		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(&pp)

		if err != nil {
			check.ExitError(err)
		}

		states := make([]int, 0, len(pp.Pipelines))

		// Check status for each pipeline
		var summary strings.Builder

		for name, pipe := range pp.Pipelines {
			inflightEvents := pipe.Events.In - pipe.Events.Out

			summary.WriteString("\n \\_")
			if thresholds.Critical.DoesViolate(float64(inflightEvents)) {
				states = append(states, check.Critical)
				summary.WriteString(fmt.Sprintf("[CRITICAL] inflight_events_%s:%d;", name, inflightEvents))
			} else if thresholds.Warning.DoesViolate(float64(inflightEvents)) {
				states = append(states, check.Warning)
				summary.WriteString(fmt.Sprintf("[WARNING] inflight_events_%s:%d;", name, inflightEvents))
			} else {
				states = append(states, check.OK)
				summary.WriteString(fmt.Sprintf("[OK] inflight_events_%s:%d;", name, inflightEvents))
			}

			// Generate perfdata for each event
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("pipelines.%s.events.in", name),
				Uom:   "c",
				Value: pipe.Events.In})
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("pipelines.%s.events.out", name),
				Uom:   "c",
				Value: pipe.Events.Out})
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("inflight_events_%s", name),
				Warn:  thresholds.Warning,
				Crit:  thresholds.Critical,
				Value: inflightEvents})
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("pipelines.%s.reloads.failures", name),
				Value: pipe.Reloads.Failures})
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("pipelines.%s.reloads.successes", name),
				Value: pipe.Reloads.Successes})
		}

		// Validate the various subchecks and use the worst state as return code
		switch result.WorstState(states...) {
		case 0:
			rc = check.OK
			output = "Inflight events alright"
		case 1:
			rc = check.Warning
			output = "Inflight events may not be alright"
		case 2:
			rc = check.Critical
			output = "Inflight events not alright"
		default:
			rc = check.Unknown
			output = "Inflight events status unknown"
		}

		check.ExitRaw(rc, output, summary.String(), "|", perfList.String())
	},
}

var pipelineReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Checks the reload configuration status of the Logstash Pipelines",
	Long:  `Checks the reload configuration status of the Logstash Pipelines`,
	Example: `
	$ check_logstash pipeline reload
	OK - Configuration successfully reloaded
	 \_[OK] Configuration successfully reloaded for pipeline Foobar for on 2021-01-01T02:07:14Z

	$ check_logstash pipeline reload --pipeline Example
	CRITICAL - Configuration reload failed
	 \_[CRITICAL] Configuration reload for pipeline Example failed on 2021-01-01T02:07:14Z`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			output string
			rc     int
			pp     logstash.Pipeline
		)

		// Creating an client and connecting to the API
		c := cliConfig.NewClient()
		// localhost:9600/_node/stats/pipelines/ will return all Pipelines
		// localhost:9600/_node/stats/pipelines/foo will return the foo Pipeline
		u, _ := url.JoinPath(c.Url, "/_node/stats/pipelines", cliPipelineConfig.PipelineName)
		resp, err := c.Client.Get(u)

		if err != nil {
			check.ExitError(err)
		}

		if resp.StatusCode != http.StatusOK {
			check.ExitError(fmt.Errorf("Could not get %s - Error: %d", u, resp.StatusCode))
		}

		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(&pp)

		if err != nil {
			check.ExitError(err)
		}

		states := make([]int, 0, len(pp.Pipelines))

		// Check the reload configuration status for each pipeline
		var summary strings.Builder

		for name, pipe := range pp.Pipelines {
			// Check Reload Timestamp
			if pipe.Reloads.LastSuccessTime != "" {
				// We could do the parsing during the unmarshall
				lastSuccessReload, errSu := time.Parse(time.RFC3339, pipe.Reloads.LastSuccessTime)
				lastFailureReload, errFa := time.Parse(time.RFC3339, pipe.Reloads.LastFailureTime)

				if errSu != nil || errFa != nil {
					states = append(states, check.Unknown)
					summary.WriteString(fmt.Sprintf("[UNKNOWN] Configuration reload for pipeline %s unknown;", name))
					summary.WriteString("\n  \\_")
					continue
				}

				summary.WriteString("\n  \\_")
				if lastFailureReload.After(lastSuccessReload) {
					states = append(states, check.Critical)
					summary.WriteString(fmt.Sprintf("[CRITICAL] Configuration reload for pipeline %s failed on %s;", name, lastFailureReload))
				} else {
					states = append(states, check.OK)
					summary.WriteString(fmt.Sprintf("[OK] Configuration successfully reloaded for pipeline %s for on %s;", name, lastSuccessReload))
				}
			}
		}

		// Validate the various subchecks and use the worst state as return code
		switch result.WorstState(states...) {
		case 0:
			rc = check.OK
			output = "Configuration successfully reloaded"
		case 1:
			rc = check.Warning
			output = "Configuration reload may not be successful"
		case 2:
			rc = check.Critical
			output = "Configuration reload failed"
		default:
			rc = check.Unknown
			output = "Configuration reload status unknown"
		}

		check.ExitRaw(rc, output, summary.String())
	},
}

var pipelineFlowCmd = &cobra.Command{
	Use:   "flow",
	Short: "Checks the flow metrics of the Logstash Pipelines",
	Long:  `Checks the flow metrics of the Logstash Pipelines`,
	Example: `
	$ check_logstash pipeline flow --warning 5 --critical 10
	OK - Flow metrics alright
	 \_[OK] queue_backpressure_example:0.34;

	$ check_logstash pipeline flow --pipeline example --warning 5 --critical 10
	CRITICAL - Flow metrics not alright
	 \_[CRITICAL] queue_backpressure_example:11.23;`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			output     string
			rc         int
			thresholds PipelineThreshold
			pp         logstash.Pipeline
			perfList   perfdata.PerfdataList
		)

		// Parse the thresholds into a central var since we need them later
		thresholds, err := parsePipeThresholds(cliPipelineConfig)
		if err != nil {
			check.ExitError(err)
		}

		// Creating an client and connecting to the API
		c := cliConfig.NewClient()
		// localhost:9600/_node/stats/pipelines/ will return all Pipelines
		// localhost:9600/_node/stats/pipelines/foo will return the foo Pipeline
		u, _ := url.JoinPath(c.Url, "/_node/stats/pipelines", cliPipelineConfig.PipelineName)
		resp, err := c.Client.Get(u)

		if err != nil {
			check.ExitError(err)
		}

		if resp.StatusCode != http.StatusOK {
			check.ExitError(fmt.Errorf("Could not get %s - Error: %d", u, resp.StatusCode))
		}

		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(&pp)

		if err != nil {
			check.ExitError(err)
		}

		states := make([]int, 0, len(pp.Pipelines))

		// Check the flow metrics for each pipeline
		var summary strings.Builder

		for name, pipe := range pp.Pipelines {
			summary.WriteString("\n \\_")
			if thresholds.Critical.DoesViolate(pipe.Flow.QueueBackpressure.Current) {
				states = append(states, check.Critical)
				summary.WriteString(fmt.Sprintf("[CRITICAL] queue_backpressure_%s:%.2f;", name, pipe.Flow.QueueBackpressure.Current))
			} else if thresholds.Warning.DoesViolate(pipe.Flow.QueueBackpressure.Current) {
				states = append(states, check.Warning)
				summary.WriteString(fmt.Sprintf("[WARNING] queue_backpressure_%s:%.2f;", name, pipe.Flow.QueueBackpressure.Current))
			} else {
				states = append(states, check.OK)
				summary.WriteString(fmt.Sprintf("[OK] queue_backpressure_%s:%.2f;", name, pipe.Flow.QueueBackpressure.Current))
			}

			// Generate perfdata for each event
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("pipelines.queue_backpressure_%s", name),
				Warn:  thresholds.Warning,
				Crit:  thresholds.Critical,
				Value: pipe.Flow.QueueBackpressure.Current})
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("pipelines.%s.output_throughput", name),
				Value: pipe.Flow.OutputThroughput.Current})
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("pipelines.%s.input_throughput", name),
				Value: pipe.Flow.InputThroughput.Current})
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("pipelines.%s.filter_throughput", name),
				Value: pipe.Flow.FilterThroughput.Current})
		}

		// Validate the various subchecks and use the worst state as return code
		switch result.WorstState(states...) {
		case 0:
			rc = check.OK
			output = "Flow metrics alright"
		case 1:
			rc = check.Warning
			output = "Flow metrics may not be alright"
		case 2:
			rc = check.Critical
			output = "Flow metrics not alright"
		default:
			rc = check.Unknown
			output = "Flow metrics status unknown"
		}

		check.ExitRaw(rc, output, summary.String(), "|", perfList.String())
	},
}

func init() {
	rootCmd.AddCommand(pipelineCmd)

	pipelineReloadCmd.Flags().StringVarP(&cliPipelineConfig.PipelineName, "pipeline", "P", "/",
		"Pipeline Name")

	pipelineFlowCmd.Flags().StringVarP(&cliPipelineConfig.PipelineName, "pipeline", "P", "/",
		"Pipeline Name")
	pipelineFlowCmd.Flags().StringVarP(&cliPipelineConfig.Warning, "warning", "w", "",
		"Warning threshold for queue Backpressure")
	pipelineFlowCmd.Flags().StringVarP(&cliPipelineConfig.Critical, "critical", "c", "",
		"Critical threshold for queue Backpressure")

	_ = pipelineFlowCmd.MarkFlagRequired("warning")
	_ = pipelineFlowCmd.MarkFlagRequired("critical")

	pipelineCmd.AddCommand(pipelineReloadCmd)
	pipelineCmd.AddCommand(pipelineFlowCmd)

	fs := pipelineCmd.Flags()

	// Default is / since we use this value for the URL Join
	// thus we have a trailing / as default
	fs.StringVarP(&cliPipelineConfig.PipelineName, "pipeline", "P", "/",
		"Pipeline Name")

	fs.StringVar(&cliPipelineConfig.Warning, "inflight-events-warn", "",
		"Warning threshold for inflight events to be a warning result. Use min:max for a range.")
	fs.StringVar(&cliPipelineConfig.Critical, "inflight-events-crit", "",
		"Critical threshold for inflight events to be a critical result. Use min:max for a range.")

	_ = pipelineCmd.MarkFlagRequired("inflight-events-warn")
	_ = pipelineCmd.MarkFlagRequired("inflight-events-crit")

	fs.SortFlags = false
}
