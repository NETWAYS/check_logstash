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
)

// To store the CLI parameters
type PipelineConfig struct {
	PipelineName       string
	InflightEventsWarn string
	InflightEventsCrit string
}

// To store the parsed CLI parameters
type PipelineThreshold struct {
	inflightEventsWarn *check.Threshold
	inflightEventsCrit *check.Threshold
}

var cliPipelineConfig PipelineConfig

func parsePipeThresholds(config PipelineConfig) (PipelineThreshold, error) {
	// Parses the CLI parameters
	var t PipelineThreshold

	inflightEventsWarn, err := check.ParseThreshold(config.InflightEventsWarn)
	if err != nil {
		return t, err
	}

	t.inflightEventsWarn = inflightEventsWarn

	inflightEventsCrit, err := check.ParseThreshold(config.InflightEventsCrit)
	if err != nil {
		return t, err
	}

	t.inflightEventsCrit = inflightEventsCrit

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
			summary    string
			rc         int
			states     []int
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

		// Check status for each pipeline
		for name, pipe := range pp.Pipelines {
			inflightEvents := pipe.Events.In - pipe.Events.Out

			summary += "\n \\_"
			if thresholds.inflightEventsCrit.DoesViolate(float64(inflightEvents)) {
				states = append(states, check.Critical)
				summary += fmt.Sprintf("[CRITICAL] inflight_events_%s:%d;", name, inflightEvents)
			} else if thresholds.inflightEventsWarn.DoesViolate(float64(inflightEvents)) {
				states = append(states, check.Warning)
				summary += fmt.Sprintf("[WARNING] inflight_events_%s:%d;", name, inflightEvents)
			} else {
				states = append(states, check.OK)
				summary += fmt.Sprintf("[OK] inflight_events_%s:%d;", name, inflightEvents)
			}

			// Generate perfdata for each event
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("pipelines.%s.events.in", name),
				Value: pipe.Events.In})
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("pipelines.%s.events.out", name),
				Value: pipe.Events.Out})
			perfList.Add(&perfdata.Perfdata{
				Label: fmt.Sprintf("inflight_events_%s", name),
				Warn:  thresholds.inflightEventsWarn,
				Crit:  thresholds.inflightEventsCrit,
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

		check.ExitRaw(rc, output, summary, "|", perfList.String())
	},
}

func init() {
	rootCmd.AddCommand(pipelineCmd)

	fs := pipelineCmd.Flags()

	// Default is / since we use this value for the URL Join
	// thus we have a trailing / as default
	fs.StringVarP(&cliPipelineConfig.PipelineName, "pipeline", "P", "/",
		"Pipeline Name")

	fs.StringVar(&cliPipelineConfig.InflightEventsWarn, "inflight-events-warn", "",
		"Warning threshold for inflight events to be a warning result. Use min:max for a range.")
	fs.StringVar(&cliPipelineConfig.InflightEventsCrit, "inflight-events-crit", "",
		"Critical threshold for inflight events to be a critical result. Use min:max for a range.")

	_ = pipelineCmd.MarkFlagRequired("inflight-events-warn")
	_ = pipelineCmd.MarkFlagRequired("inflight-events-crit")

	fs.SortFlags = false
}
