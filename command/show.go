package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/command/jsonplan"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
)

// ShowCommand is a Command implementation that reads and outputs the
// contents of a Terraform plan or state file.
type ShowCommand struct {
	Meta
}

func (c *ShowCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	var jsonOutput bool

	cmdFlags := flag.NewFlagSet("show", flag.ContinueOnError)
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output (only available when showing a plan")

	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 2 {
		c.Ui.Error(
			"The show command expects at most two arguments.\n The path to a " +
				"Terraform state or plan file, and optionally -json for json output.\n")
		cmdFlags.Usage()
		return 1
	}

	var diags tfdiags.Diagnostics

	// Load the backend
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// We require a local backend
	local, ok := b.(backend.Local)
	if !ok {
		c.showDiagnostics(diags) // in case of any warnings in here
		c.Ui.Error(ErrUnsupportedLocalOp)
		return 1
	}

	// the show command expects the config dir to always be the cwd
	cwd, err := os.Getwd()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting cwd: %s", err))
		return 1
	}

	// Build the operation
	opReq := c.Operation(b)
	opReq.ConfigDir = cwd
	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		c.showDiagnostics(diags)
		return 1
	}

	// Get the context
	ctx, _, ctxDiags := local.Context(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	schemas := ctx.Schemas()

	env := c.Workspace()

	var planErr, stateErr error
	var path string
	var plan *plans.Plan
	var state *states.State
	if len(args) > 0 {
		path = args[0]
		pr, err := planfile.Open(path)
		if err != nil {
			if jsonOutput == true {
				c.Ui.Error(fmt.Sprintf(
					"Error: JSON output not available for state",
				))
				return 1
			}
			f, err := os.Open(path)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Error loading file: %s", err))
				return 1
			}
			defer f.Close()

			var stateFile *statefile.File
			stateFile, err = statefile.Read(f)
			if err != nil {
				stateErr = err
			} else {
				state = stateFile.State
			}
		} else {
			plan, err = pr.ReadPlan()
			if err != nil {
				planErr = err
			}
		}
	} else {
		// Get the state
		stateStore, err := b.StateMgr(env)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
			return 1
		}

		if err := stateStore.RefreshState(); err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
			return 1
		}

		state = stateStore.State()
		if state == nil {
			c.Ui.Output("No state.")
			return 0
		}
	}

	if plan == nil && state == nil {
		c.Ui.Error(fmt.Sprintf(
			"Terraform couldn't read the given file as a state or plan file.\n"+
				"The errors while attempting to read the file as each format are\n"+
				"shown below.\n\n"+
				"State read error: %s\n\nPlan read error: %s",
			stateErr,
			planErr))
		return 1
	}

	if plan != nil {
		if jsonOutput == true {

			_, snapshot, loadDiags := opReq.ConfigLoader.LoadConfigWithSnapshot(cwd)
			if loadDiags.HasErrors() {
				c.showDiagnostics(diags)
				return 1
			}
			jsonPlan, err := jsonplan.Marshal(snapshot, plan, state)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Failed to load config: %s", err))
				return 1
			}
			c.Ui.Output(string(jsonPlan))
			return 0
		}
		dispPlan := format.NewPlan(plan.Changes)
		c.Ui.Output(dispPlan.Format(c.Colorize()))
		return 0
	}

	c.Ui.Output(format.State(&format.StateOpts{
		State:   state,
		Color:   c.Colorize(),
		Schemas: schemas,
	}))
	return 0
}

func (c *ShowCommand) Help() string {
	helpText := `
Usage: terraform show [options] [path]

  Reads and outputs a Terraform state or plan file in a human-readable
  form. If no path is specified, the current state will be shown.

Options:

  -no-color           If specified, output won't contain any color.
  -json				  If specified, output the Terraform plan in a machine-
						readable form. Only available for plan files.

`
	return strings.TrimSpace(helpText)
}

func (c *ShowCommand) Synopsis() string {
	return "Inspect Terraform state or plan"
}
