package main

import (
	"bufio"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"github.com/pkg/errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	//"runtime"
	"sync"
	"time"
	"runtime"
)

type Content struct {
	url   string
	image *image.Image
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	mstart := time.Now()

	input_file, err := os.Open("input.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer input_file.Close()

	file, err := os.Create("op.csv")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)

	urls := make(chan string)

	var io_wg sync.WaitGroup
	var proc_wg sync.WaitGroup
	numIOGroups := 10
	numProcGroups := 10
	io_sentinels := make(chan bool, numIOGroups)
	proc_sentinels := make(chan bool, numProcGroups)
	content_chan := make(chan *Content, numProcGroups)

	io_wg.Add(numIOGroups)
	proc_wg.Add(numProcGroups)

	for i := 0; i < numIOGroups; i++ {
		go func() {
			defer io_wg.Done()
			for {
				select {
				case url := <-urls:
					resp, err := http.Get(url)
					if err != nil {
						wrappedErr := errors.Wrap(err, "[1] failed with error:")
						fmt.Println(err)
						fmt.Println(wrappedErr)
						continue
					}

					defer resp.Body.Close()

					image, _, err := image.Decode(resp.Body)
					if err != nil {
						fmt.Println("decode error")
						log.Fatal(err)
					}

					content_chan <- &Content{url: url, image: &image}
				default:
					select {
					case <-io_sentinels:
						return
					default:
						time.Sleep(1 * time.Second)
						//fmt.Println("io pass")
					}
				}
			}
		}()
	}

	for i := 0; i < numProcGroups; i++ {
		go func() {
			defer proc_wg.Done()
			for {
				select {
				case content := <-content_chan:
					data := ProcessFile(content.image)

					sl := data[:]

					sl = append(sl, "")
					copy(sl[1:], sl[:])
					sl[0] = content.url
					//fmt.Println(sl)

					w.Write(sl)
				default:
					select {
					case <-proc_sentinels:
						return
					default:
						time.Sleep(1 * time.Second)
						//fmt.Println("proc pass")
					}
				}
			}
		}()
	}

	scanner := bufio.NewScanner(input_file)
	set := make(map[string]bool)

	for scanner.Scan() {
		url := scanner.Text()

		_, ok := set[url]
		if !ok {
			//set[url] = true
			urls <- scanner.Text()
		} else {
			fmt.Println("ALREADY PRESENT")
		}
	}

	for i := 0; i < numIOGroups; i++ {
		io_sentinels <- true
	}
	io_wg.Wait()

	for i := 0; i < numProcGroups; i++ {
		proc_sentinels <- true
	}
	proc_wg.Wait()

	w.Flush()

	fmt.Println("processing took ", time.Since(mstart))
}

func ProcessFile(m *image.Image) [3]string {
	PrintMemUsage()
	im := *m

	bounds := im.Bounds()

	min_x, min_y, max_x, max_y := bounds.Min.X, bounds.Min.Y,
		bounds.Max.X, bounds.Max.Y

	values := make(map[[3]uint8]int)

	for y := min_y; y < max_y; y++ {
		for x := min_x; x < max_x; x++ {
			r, g, b, _ := im.At(x, y).RGBA()
			values[[3]uint8{uint8(r>>8), uint8(g>>8), uint8(b>>8)}]++
		}
	}
	runtime.GC()

	topColorCounts := [3]int{0, 0, 0}
	topColors := [3][3]uint8{}
	for key, val := range values {
		if val >= topColorCounts[0] {
			tmp := topColorCounts[0]
			tmp2 := topColorCounts[1]

			tmpKey := topColors[0]
			tmpKey2 := topColors[1]

			topColorCounts[0] = val
			topColorCounts[1] = tmp
			topColorCounts[2] = tmp2

			topColors[0] = key
			topColors[1] = tmpKey
			topColors[2] = tmpKey2
		} else if val >= topColorCounts[1] {
			tmp := topColorCounts[1]
			tmpKey := topColors[1]

			topColorCounts[1] = val
			topColorCounts[2] = tmp

			topColors[1] = key
			topColors[2] = tmpKey
		} else if val > topColorCounts[2] {
			topColorCounts[2] = val
			topColors[2] = key
		}
	}

	topColorStrings := [3]string{}
	for idx, l := range topColors {
		topColorStrings[idx] = fmt.Sprintf("#%02X%02X%02X", l[0], l[1], l[2])
	}

	return topColorStrings
}

func PrintMemUsage() {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        // For info on each, see: https://golang.org/pkg/runtime/#MemStats
        fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
        fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
        fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
        fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}