package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
	"github.com/tokuhirom/go-hsperfdata/attach"
)

type Pids []*process.Process

func main() {
	pid, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal("need a valid pid")
	}
	sock, err := attach.New(pid)
	if err != nil {
		log.Fatalf("unable to attach: %v", err)
	}
	err = sock.Execute("threaddump")
	if err != nil {
		log.Fatalf("cannot write to unix socket: %s", err)
	}

	stack, err := sock.ReadString()
	if err != nil {
		log.Fatalf("unable to read from socket: %v", err)
	}
	jinfo := getInfo(stack)

	p, err := process.NewProcess(int32(pid))
	if err != nil {
		log.Fatal(err)
	}
	threads, err := p.Threads()
	if err != nil {
		log.Fatal("cant find threads")
	}
	u1, err := getcpu(threads)
	if err != nil {
		log.Fatalf("unable to get cpu info: %v", err)
	}

	for n, i := range jinfo {
		if v, ok := u1[int(i.Pid)]; ok {
			jinfo[n].Cpu = v
		}
	}

	sort.SliceStable(jinfo, func(i, j int) bool {
		return jinfo[i].Cpu > jinfo[j].Cpu
	})
	for _, i := range jinfo {
		fmt.Printf("thread %d, nid %s, name [%s], cpu %f\n", i.Pid, i.Nid, i.Name, i.Cpu)
	}

}

// takex a pid and returns a nid
func convertPid(p int32) string {
	//return fmt.Sprintf("%x", p)
	return strconv.FormatInt(int64(p), 16)
}

//takes an nid and returns a pid
func convertNid(s string) (int32, error) {
	n, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return int32(0), err
	}
	return int32(n), nil
}
func getcpu(threads map[int32]*cpu.TimesStat) (map[int]float64, error) {
	dm := make(map[int]float64, 0)
	for k, v := range threads {
		val, err := cpuusage(int32(k), v)
		if err == nil {
			dm[int(k)] = val
		}
	}
	return dm, nil

}

// get cpu utilization for threads
func ThreadCPUUsage(pids Pids) (map[int]map[int]float64, error) {
	m := make(map[int]map[int]float64)
	for _, p := range pids {

		proc, err := process.NewProcess(int32(p.Pid))
		if err != nil {
			return nil, fmt.Errorf("unable to determine threads for %d, %v", p.Pid, err)
		}

		threads, err := proc.Threads()
		if err != nil {
			return nil, fmt.Errorf("unable to determine thread info for %d: %v", p.Pid, err)
		}
		dm := make(map[int]float64, 0)

		m[int(p.Pid)] = dm
		for k, v := range threads {
			val, err := cpuusage(int32(k), v)
			if err == nil {
				dm[int(k)] = val
			}
		}

	}
	return m, nil
}

// get cpu utilization for threads
func cpuusage(t int32, v *cpu.TimesStat) (float64, error) {
	proc, err := process.NewProcess(t)
	if err != nil {
		return 0.0, err
	}

	ct, err := proc.CreateTime()
	if err != nil {
		return 0.0, err
	}

	//cput := v.Total()
	created := time.Unix(0, ct*int64(time.Millisecond))
	runtime := time.Since(created).Seconds()
	if runtime <= 0 {
		return 0, nil
	}

	// sysconf(_SC_CLK_TCK) = 100 // see below
	return 100 * v.Total() / runtime, nil
}

type Jinfo struct {
	Nid    string
	Name   string
	Status string
	Pid    int32
	Value  string
	Cpu    float64
}

func getInfo(s string) []Jinfo {
	p := makePara(s)
	jinfo := makeJinfo(p)
	return *jinfo
}

/*
func main() {
	b, err := ioutil.ReadFile("paragraph.txt")
	if err != nil {
		log.Fatal(err)
	}
	p := makePara(string(b))
	fmt.Println(len(p))

	jinfo := makeJinfo(p)
	for _, i := range *jinfo {
		fmt.Printf("================>\n Nid: %s\nStatus: %s\nPid: %d\nValue:%s\n =================\n\n", i.Nid, i.Status, i.Pid, i.Value)
	}
}
*/

func makeJinfo(p []string) *[]Jinfo {
	m := make([]Jinfo, 0)
	for _, para := range p {
		lines := strings.Split(para, "\n")
		nid, err := extractNid(lines[0])
		if err != nil {
			fmt.Printf("line %s does not have nid, skipping\n", lines[0])
			continue
		}
		name, err := extractName(lines[0])
		if err != nil {
			fmt.Printf("line %s does not have  a name, skipping\n", lines[0])
		}
		status, err := extractStatus(lines[1])
		if err != nil {
			fmt.Printf("line %s does not have status,err %v, skipping\n", lines[1], err)
		}
		j := Jinfo{
			Nid:    nid,
			Name:   name,
			Value:  para,
			Status: status,
			Pid:    nid2pid(nid),
		}
		m = append(m, j)

	}
	return &m
}

func nid2pid(s string) int32 {
	n, err := convertNid(s)
	if err != nil {
		return 0
	}
	return int32(n)
}
func extractName(l string) (string, error) {
	fields := strings.Split(l, `"`)
	if len(fields) > 1 && fields[0] == "" && fields[1] != "" {
		return fields[1], nil
	}
	return "", fmt.Errorf("invalid string found")

}
func extractNid(l string) (string, error) {
	fields := strings.Fields(l)
	for _, field := range fields {
		f := strings.Split(field, "nid=0x")
		if f[0] == "" && f[1] != "" {
			return f[1], nil
		}
	}
	return "", fmt.Errorf("no field found")
}

func extractStatus(l string) (string, error) {
	if !strings.Contains(l, "java.lang.Thread.State") {
		return "", fmt.Errorf("no threadstate found")
	}
	f := strings.Split(l, "java.lang.Thread.State: ")
	if len(f) != 2 {
		return "", fmt.Errorf("no state found")
	}
	status := strings.TrimSpace(f[1])
	if status == "" {
		return "", fmt.Errorf("no result")
	}
	return status, nil
}
func makePara(s string) []string {
	var l string
	var paras []string
	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			paras = append(paras, l)
			l = ""
		} else {
			l = l + line + "\n"
		}
	}
	return paras
}
