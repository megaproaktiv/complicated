package main

import (
	"github.com/alecthomas/kong"
	"github.com/megaproaktiv/antipsychotic/checker/config"
	"github.com/megaproaktiv/antipsychotic/checker/reasoning"
)

var CLI struct {
	Short bool `short:"s" long:"short" help:"Run short questions"`
	Long  bool `short:"l" long:"long" help:"Run long questions"`
}

func main() {
	ctx := kong.Parse(&CLI, kong.Name("checker"),
		kong.Description("Run Amazon Bedrock Guardrail- automated reasoning checks"))
	config.Setup()

	if CLI.Short {
		reasoning.RunShort()
	} else if CLI.Long {
		reasoning.RunLong()
	} else {
		ctx.Printf("Use -s/--short or -l/--long to specify question type\n")
	}
}
