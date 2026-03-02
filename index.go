package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/elastic/go-sysinfo"
	"github.com/joho/godotenv"
	disk "github.com/limesdotpink/minio-disk"
	"github.com/mattn/go-mastodon"
)

func main() {
	host, err := sysinfo.Host()
	var tootText string

	godotenv.Load()

	if err != nil {
		log.Fatal("cannot initialize go-sysinfo. catastrophic failure")
		return
	}

	cu, cuerr := getCpuUsage()
	if cuerr == nil {
		tootText += "CPU\n"
		tootText += progressBar(cu)
	} else {
		fmt.Printf("CPU load unsupported (currently linux only)")
	}

	meminfo, merr := host.Memory()
	if merr == nil {
		memFree := meminfo.Total - meminfo.Available
		tootText += fmt.Sprintf("\n\nRAM: %s/%s\n", humanize.Bytes(memFree), humanize.Bytes(meminfo.Total))
		tootText += progressBar((float64(memFree) / float64(meminfo.Total) * 100))

		virtFree := meminfo.VirtualTotal - meminfo.VirtualFree
		tootText += fmt.Sprintf("\nSwap: %s/%s\n", humanize.Bytes(virtFree), humanize.Bytes(meminfo.VirtualTotal))
		tootText += progressBar((float64(virtFree) / float64(meminfo.VirtualTotal) * 100))
	}

	samedrive, root, rerr, home, herr := getDiskInfo()
	if rerr == nil {
		rf, rt, rp := parseDiskUsage(root)

		if samedrive || herr != nil {
			tootText += fmt.Sprintf("\n\nDisk (/): %s/%s\n", rf, rt)
			tootText += progressBar(rp)
		} else {
			tootText += "\n\nDisks:"
			rf, rt, rp := parseDiskUsage(root)
			tootText += fmt.Sprintf("\n/: %s/%s\n", rf, rt)
			tootText += progressBar(rp)

			hf, ht, rp := parseDiskUsage(home)
			tootText += fmt.Sprintf("\n/home: %s/%s\n", hf, ht)
			tootText += progressBar(rp)
		}
	}

	tootText += fmt.Sprintf("\n\nUptime: %s\n", formatUptime(host.Info().Uptime().Round(time.Second).String()))

	fmt.Print(tootText)

	postToot(tootText)
}

func getDiskInfo() (bool, disk.Info, error, disk.Info, error) {
	rootinfo, rerr := disk.GetInfo("/", true)
	homeinfo, herr := disk.GetInfo("/home", true)

	return rootinfo.Major == homeinfo.Major && rootinfo.Minor == homeinfo.Minor, rootinfo, rerr, homeinfo, herr
}

func parseDiskUsage(diskinfo disk.Info) (string, string, float64) {
	return humanize.Bytes(diskinfo.Used), humanize.Bytes(diskinfo.Total), float64(diskinfo.Used) / float64(diskinfo.Total) * 100
}

// START CC BY-SA 3.0 https://stackoverflow.com/a/17783687
func getCpuUsage() (float64, error) {
	idle0, total0, err1 := getCPUSample()
	time.Sleep(3 * time.Second)
	idle1, total1, _ := getCPUSample()

	if err1 != nil {
		return 0, err1
	}

	idleTicks := float64(idle1 - idle0)
	totalTicks := float64(total1 - total0)
	cpuUsage := 100 * (totalTicks - idleTicks) / totalTicks

	return cpuUsage, err1

}

func getCPUSample() (idle, total uint64, err error) {
	contents, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == "cpu" {
			numFields := len(fields)
			for i := 1; i < numFields; i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					fmt.Println("Error: ", i, fields[i], err)
				}
				total += val // tally up all the numbers to get total ticks
				if i == 4 {  // idle is the 5th field in the cpu line
					idle = val
				}
			}
			return
		}
	}
	return
}

// END CC BY-SA 3.0 https://stackoverflow.com/a/17783687

type uptime struct {
	d uint32
	h uint8
	m uint8
	s uint8
}

func formatUptime(raw string) string {
	var up uptime
	var rawh uint32

	fmt.Sscanf(raw, "%dh%dm%ds", &rawh, &up.m, &up.s)

	up.d = rawh / 24
	up.h = uint8(rawh % 24)

	return fmt.Sprintf("%dd, %dh, %dm, %ds", up.d, up.h, up.m, up.s)
}

func progressBar(perc float64) string {
	var barWidth float64 = 20
	var bar string
	r := int(math.Round(perc / 100 * barWidth))

	bar = fmt.Sprintf("[%s%s] %.1f%%", strings.Repeat("￭", r), strings.Repeat("･", int(barWidth)-r), perc)

	return bar
}

func postToot(t string) {
	config := &mastodon.Config{
		Server:       os.Getenv("INSTANCE_URL"),
		ClientID:     os.Getenv("BOT_CLIENT_KEY"),
		ClientSecret: os.Getenv("BOT_CLIENT_SECRET"),
		AccessToken:  os.Getenv("BOT_ACCESS_TOKEN"),
	}

	c := mastodon.NewClient(config)

	toot := mastodon.Toot{
		Status:     t,
		Visibility: "public",
	}

	post, err := c.PostStatus(context.Background(), &toot)
	if err == nil {
		fmt.Printf("\n\n%s\n", post.URL)
	}
}
