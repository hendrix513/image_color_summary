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
)

type Content struct {
	url   string
	image *image.Image
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	mstart := time.Now()
	//runtime.GOMAXPROCS(1)

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
	numIOGroups := 8
	numProcGroups := 20
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
					//start := time.Now()
					resp, err := http.Get(url)
					if err != nil {
						wrappedErr := errors.Wrap(err, "[1] failed with error:")
						fmt.Println(err)
						fmt.Println(wrappedErr)
						continue
					}

					//dur := time.Since(start)
					//size := (max_y-min_y)*(max_x-min_x)
					//fmt.Println("processing url took ", dur)
					//start := time.Now()

					defer resp.Body.Close()

					image, _, err := image.Decode(resp.Body)
					if err != nil {
						fmt.Println("decode error")
						log.Fatal(err)
					}

					//dur = time.Since(start)
					//size := (max_y-min_y)*(max_x-min_x)
					//fmt.Println("getting image took ", dur)

					//start = time.Now()
					content_chan <- &Content{url: url, image: &image}
					//dur = time.Since(start)
					//size := (max_y-min_y)*(max_x-min_x)
					//fmt.Println("putting on channel took ", dur)
				default:
					select {
					case <-io_sentinels:
						return
					default:
						time.Sleep(2 * time.Second)
						fmt.Println("io pass")
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
					fmt.Println(sl)

					w.Write(sl)
				default:
					select {
					case <-proc_sentinels:
						return
					default:
						time.Sleep(2 * time.Second)
						fmt.Println("proc pass")
					}
				}
			}
		}()
	}

	scanner := bufio.NewScanner(input_file)

	for scanner.Scan() {
		urls <- scanner.Text()
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
	im := *m

	//start := time.Now()
	// Get the data

	bounds := im.Bounds()

	min_x, min_y, max_x, max_y := bounds.Min.X, bounds.Min.Y,
		bounds.Max.X, bounds.Max.Y

	values := make(map[string]int)

	for y := min_y; y < max_y; y++ {
		for x := min_x; x < max_x; x++ {
			r, g, b, a := im.At(x, y).RGBA()
			h := RGBAToHex(uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8))
			values[h]++
		}
	}

	topColorCounts := [3]int{0, 0, 0}
	//topColors := [3]string{"", "", ""}
	topColors := [3]string{}
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

	//dur := time.Since(start)
	//size := (max_y-min_y)*(max_x-min_x)
	//fmt.Println("processing img took ", dur, size, int(dur)/size)
	return topColors
}

func RGBAToHex(r, g, b, a uint8) string {
	if a == 255 {
		return fmt.Sprintf("#%02X%02X%02X", r, g, b)
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", r, g, b, a)
}
