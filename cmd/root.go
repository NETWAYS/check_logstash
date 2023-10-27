package cmd

import (
	"os"

	"github.com/NETWAYS/go-check"
	"github.com/spf13/cobra"
)

var Timeout = 30

var rootCmd = &cobra.Command{
	Use:   "check_logstash",
	Short: "An Icinga check plugin to check Logstash",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		go check.HandleTimeout(Timeout)
	},
	Run: Usage,
}

func Execute(version string) {
	defer check.CatchPanic()

	rootCmd.Version = version
	rootCmd.VersionTemplate()

	if err := rootCmd.Execute(); err != nil {
		check.ExitError(err)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.DisableAutoGenTag = true

	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})

	pfs := rootCmd.PersistentFlags()
	pfs.StringVarP(&cliConfig.Hostname, "hostname", "H", "localhost",
		"Hostname of the Logstash server (CHECK_LOGSTASH_HOSTNAME)")
	pfs.IntVarP(&cliConfig.Port, "port", "p", 9600,
		"Port of the Logstash server")
	pfs.BoolVarP(&cliConfig.Secure, "secure", "s", false,
		"Use a HTTPS connection")
	pfs.BoolVarP(&cliConfig.Insecure, "insecure", "i", false,
		"Skip the verification of the server's TLS certificate")
	pfs.StringVarP(&cliConfig.Bearer, "bearer", "b", "",
		"Specify the Bearer Token for server authentication (CHECK_LOGSTASH_BEARER)")
	pfs.StringVarP(&cliConfig.BasicAuth, "user", "u", "",
		"Specify the user name and password for server authentication <user:password> (CHECK_LOGSTASH_BASICAUTH)")
	pfs.StringVarP(&cliConfig.CAFile, "ca-file", "", "",
		"Specify the CA File for TLS authentication (CHECK_LOGSTASH_CA_FILE)")
	pfs.StringVarP(&cliConfig.CertFile, "cert-file", "", "",
		"Specify the Certificate File for TLS authentication (CHECK_LOGSTASH_CERT_FILE)")
	pfs.StringVarP(&cliConfig.KeyFile, "key-file", "", "",
		"Specify the Key File for TLS authentication (CHECK_LOGSTASH_KEY_FILE)")
	pfs.IntVarP(&Timeout, "timeout", "t", Timeout,
		"Timeout in seconds for the CheckPlugin")

	rootCmd.Flags().SortFlags = false
	pfs.SortFlags = false

	help := rootCmd.HelpTemplate()
	rootCmd.SetHelpTemplate(help + Copyright)

	check.LoadFromEnv(&cliConfig)
}

func Usage(cmd *cobra.Command, _ []string) {
	_ = cmd.Usage()

	os.Exit(3)
}
