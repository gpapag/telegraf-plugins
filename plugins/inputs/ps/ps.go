package ps

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/plugins/inputs"
)

const (
	processSelection = `-axo`
	infoSelection    = `pid=,ppid=,comm=,args=,nlwp=,rss=,vsz=,%mem=,%cpu=,psr=,ruser=,stat=`
	fieldName        = `ps`
	tag              = `ps`
)

type psInfo struct {
	Pid   int     `json:"pid"`
	Ppid  int     `json:"ppid"`
	Comm  string  `json:"command"`
	Args  string  `json:"args"`
	Nlwp  int     `json:"threads"`
	Rss   int     `json:"rss"`
	Vsz   int     `json:"vsize"`
	Mem   float64 `json:"mem"`
	CPU   float64 `json:"cpu"`
	Psr   int     `json:"processor"`
	Ruser string  `json:"user"`
	Stat  string  `json:"status"`
}

// PS executes a ps command to collect information about the processes
// running on the host.
//
type PS struct {
	procSelection string
	infoSelection string
	Timeout       internal.Duration
}

// init initializes the package.
func init() {
	inputs.Add("ps", func() telegraf.Input {
		return newPS(processSelection, infoSelection)
	})
}

// newPS returns a pointer to a new PS object.
func newPS(processSelection string, infoSelection string) *PS {
	return &PS{
		procSelection: processSelection,
		infoSelection: infoSelection,
		Timeout:       internal.Duration{Duration: time.Second * 5},
	}
}

// Description returns a short description about the plugin.
func (p *PS) Description() string {
	return "Read information about the processes running on the host."
}

// SampleConfig returns a sample configuration for the plugin.
func (p *PS) SampleConfig() string {
	return `
	## Timeout for command to complete.
	#timeout = "5s"
	`
}

// Gather parses the output of the ps command and stores the output in
// the accumulator acc.
func (p *PS) Gather(acc telegraf.Accumulator) error {
	psCommand := strings.Join([]string{"/bin/ps", p.procSelection, p.infoSelection}, " ")
	jsonArray, err := p.processCommand(psCommand)
	if err != nil {
		acc.AddError(err)
		return fmt.Errorf("ps: unable to gather metrics: %s", err)
	}

	metric, err := metric.New(
		fieldName,
		map[string]string{"plugin": tag},
		map[string]interface{}{"fields": string(jsonArray)},
		time.Now().UTC())
	if err != nil {
		acc.AddError(err)
		return fmt.Errorf("ps: unable to gather metrics: %s", err)
	}

	acc.AddFields(metric.Name(), metric.Fields(), metric.Tags(), metric.Time())
	return nil
}

// processCommand executes the command and returns a slice of json objects
// containing the results.
func (p *PS) processCommand(command string) ([]byte, error) {
	var err error

	var splitCmd []string
	splitCmd, err = shellquote.Split(command)
	if err != nil || len(splitCmd) == 0 {
		return nil, err
	}

	var out bytes.Buffer
	cmd := exec.Command(splitCmd[0], splitCmd[1:]...)
	cmd.Stdout = &out
	if err := internal.RunTimeout(cmd, p.Timeout.Duration); err != nil {
		return nil, err
	}

	var jsonInfo []psInfo
	jsonInfo, err = p.parse(out.String())
	if err != nil {
		return nil, err
	}

	var jsonArray []byte
	jsonArray, err = json.Marshal(jsonInfo)
	if err != nil {
		return nil, err
	}

	return jsonArray, nil
}

// parse returns a slice of json objects based on the text in out.
func (p *PS) parse(in string) ([]psInfo, error) {
	var parser = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s+(.+?)\s+(.+?)\s+(\d+)\s+(\d+)\s+(\d+\.\d+)\s+(\d+\.\d+)\s+(.+?)\s+(.+?)$`)

	var psInfoArray []psInfo
	scanner := bufio.NewScanner(strings.NewReader(in))
	for scanner.Scan() {
		results := parser.FindAllStringSubmatch(scanner.Text(), -1)
		if results == nil {
			continue
		}
		psInfoElement, err := p.parseLine(results)
		if err != nil {
			continue
		}
		psInfoArray = append(psInfoArray, *psInfoElement)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return psInfoArray, nil
}

// parseLine returns a psInfo struct with the information of a single process.
func (p *PS) parseLine(results [][]string) (*psInfo, error) {
	var err error

	var pid int
	pid, err = strconv.Atoi(results[0][1])
	if err != nil {
		return nil, err
	}
	var ppid int
	ppid, err = strconv.Atoi(results[0][2])
	if err != nil {
		return nil, err
	}
	command := results[0][3]
	args := results[0][4]
	var rss int
	rss, err = strconv.Atoi(results[0][5])
	if err != nil {
		return nil, err
	}
	var vsize int
	vsize, err = strconv.Atoi(results[0][6])
	if err != nil {
		return nil, err
	}
	var mem float64
	mem, err = strconv.ParseFloat(results[0][7], 64)
	if err != nil {
		return nil, err
	}
	var cpu float64
	cpu, err = strconv.ParseFloat(results[0][8], 64)
	if err != nil {
		return nil, err
	}
	user := results[0][9]
	status := results[0][10]

	return &psInfo{
		pid,
		ppid,
		command,
		args,
		rss,
		vsize,
		mem,
		cpu,
		user,
		status,
	}, nil
}
