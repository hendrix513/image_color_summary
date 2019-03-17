package main

import (
	"bufio"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	input_file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer input_file.Close()

	file, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)

	var io_wg sync.WaitGroup
	numIOGroups := 16
	io_sentinels := make(chan bool, numIOGroups)
	urls := make(chan string, numIOGroups)

	io_wg.Add(numIOGroups)

	for i := 0; i < numIOGroups; i++ {
		go func() {
			defer io_wg.Done()
			for {
				select {
				case url := <-urls:
				        defer runtime.GC()
					resp, err := http.Get(url)
					if err != nil {
						log.Fatal(err)
						continue
					}

					defer resp.Body.Close()

					image, _, err := image.Decode(resp.Body)
					if err != nil {
						log.Println(err)
						continue
					}

					w.Write(ProcessFile(&image, url))
				default:
					select {
					case <-io_sentinels:
						return
					default:
						time.Sleep(time.Second)
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
			set[url] = true
			urls <- scanner.Text()
		}
	}

	for i := 0; i < numIOGroups; i++ {
		io_sentinels <- true
	}
	io_wg.Wait()

	w.Flush()
}

func ProcessFile(m *image.Image, url string) []string {
	im := *m

	bounds := im.Bounds()

	min_x, min_y, max_x, max_y := bounds.Min.X, bounds.Min.Y,
		bounds.Max.X, bounds.Max.Y

	values := make(map[[3]uint8]int)

	for y := min_y; y < max_y; y++ {
		for x := min_x; x < max_x; x++ {
			r, g, b, _ := im.At(x, y).RGBA()
			values[[3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}]++
		}
	}

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

	topColorStrings := [4]string{url, "", "", ""}
	for idx, l := range topColors {
		topColorStrings[idx+1] = fmt.Sprintf("#%02X%02X%02X", l[0], l[1], l[2])
	}

	return topColorStrings[:]
}
