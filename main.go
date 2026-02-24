// fixBadDisk 坏块屏蔽工具 防作弊测速工具
// Golang重构 Sparkle 20260225

package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const ver = "14.0"

// fileGenResult 预生成文件的结果
type fileGenResult struct {
	data []byte
	name string
}

// fileGenerator 使用 worker pool 并发预生成随机文件
type fileGenerator struct {
	fsize   float64
	ch      chan fileGenResult
	done    chan struct{}
	wg      sync.WaitGroup
	workers int
}

func newFileGenerator(fsize float64, bufSize int) *fileGenerator {
	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	return &fileGenerator{
		fsize:   fsize,
		ch:      make(chan fileGenResult, bufSize),
		done:    make(chan struct{}),
		workers: workers,
	}
}

func (g *fileGenerator) start() {
	for i := 0; i < g.workers; i++ {
		g.wg.Add(1)
		go func() {
			defer g.wg.Done()
			for {
				data, ok := genFile(g.fsize)
				if !ok {
					return
				}
				select {
				case g.ch <- fileGenResult{data: data, name: data2name(data)}:
				case <-g.done:
					return
				}
			}
		}()
	}
}

func (g *fileGenerator) stop() {
	close(g.done) // 通知所有 worker 退出
	g.wg.Wait()   // 等待全部退出后再返回
}

// genFile 生成指定大小的随机数据，返回 (data, ok)
func genFile(fsize float64) ([]byte, bool) {
	size := int(1024 * 1024 * fsize)
	if size <= 0 {
		return nil, false
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return nil, false
	}
	return buf, true
}

func data2name(data []byte) string {
	h := md5.Sum(data)
	return hex.EncodeToString(h[:])[:8]
}

func getFreeSpaceMB(folder string) float64 {
	return getFreeSpaceMBOS(folder)
}

func connectErr(badDir string) bool {
	cwd, err := os.Getwd()
	if err != nil {
		return true
	}
	if cwd == badDir {
		if runtime.GOOS == "windows" {
			_ = os.Chdir("C:/")
		} else {
			_ = os.Chdir("/")
		}
	}
	if err := os.Chdir(badDir); err != nil {
		return true
	}
	return false
}

func formatDuration(secs float64) string {
	h := int(secs) / 3600
	m := (int(secs) % 3600) / 60
	s := int(secs) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func printProgress(minsp, maxsp, avg, minavg, done, total, elapsed, remaining, nsp, nt float64) {
	fmt.Printf("\033[F\033[KMin: %.3fM/s Max: %.3fM/s Avg: %.3fM/s MinAvg: %.3fM/s\n%.3fM/%.3fM %s/%s (%.3fM/s %.6fs)  ",
		minsp, maxsp, avg, minavg,
		done, total,
		formatDuration(elapsed), formatDuration(remaining),
		nsp, nt,
	)
}

func saveSpeedLine(per int, minsp, maxsp, avg, minavg, nsp, nt float64) string {
	return fmt.Sprintf("%d%% Min: %.3fM/s Max: %.3fM/s Avg: %.3fM/s MinAvg: %.3fM/s (%.3fM/s %.6fs)",
		per, minsp, maxsp, avg, minavg, nsp, nt)
}

func main() {
	const savePer = 0.1
	fsize := 10.0
	setSize := false

	doTest := fileExists("bad") && fileExists("fixBadDiskWriteOK.txt")
	doWrite := !fileExists("fixBadDiskWriteOK.txt")

	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help":
			fmt.Printf("一键 u盘/内存卡/硬盘 坏块/坏道 维修工具 防作弊测速工具 v%s\n", ver)
			fmt.Println("Usage: fixBadDisk [filesize] [w|r|rw] [maxsize]")
			fmt.Println("  -h, --help: 显示当前帮助信息")
			fmt.Println("  filesize: 单个文件大小，fat32下最大为4096M，且最多33000个文件")
			fmt.Println("  w: 写入测试")
			fmt.Println("  t或r: 读测试")
			fmt.Println("  rw: 写满后马上读，可能出现误差，不建议非大容量机械硬盘使用")
			fmt.Println("  maxsize: 最大写入量，用于写入测速时指定大小")
			fmt.Println("输出：\nMin: 最小速度 Max: 最大速度 Avg: 平均速度\n已写入/总容量 已用时间/剩余时间 (当前速度 当前用时)")
			fmt.Printf("默认%.0fM，写入满后退出，重新拔插再运行将测试，举个栗子：\n", fsize)
			fmt.Println("测试4k读速度 fixBadDisk 0.004 w 100")
			fmt.Println("测试4k写速度 fixBadDisk 0.004 r")
			fmt.Println("Press Enter to exit 按回车退出")
			fmt.Scanln()
			return
		case "w", "-w":
			doWrite = true
			doTest = false
		case "r", "-r", "t", "-t":
			doWrite = false
			doTest = true
		case "rw", "-rw":
			doWrite = true
			doTest = true
			fmt.Println("写满后马上读，可能出现误差，不建议非大容量机械硬盘使用")
		default:
			s := strings.ToLower(args[0])
			s = strings.TrimSuffix(s, "b")
			s = strings.TrimSuffix(s, "k")
			s = strings.TrimSuffix(s, "m")
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				fsize = v
				setSize = true
			}
		}
	}

	if len(args) > 1 {
		switch args[1] {
		case "w", "-w":
			doWrite = true
			doTest = false
		case "r", "-r", "t", "-t":
			doWrite = false
			doTest = true
		case "rw", "-rw":
			doWrite = true
			doTest = true
			fmt.Println("写满后马上读，可能出现误差，不建议非大容量机械硬盘使用")
		default:
			s := strings.ToLower(args[1])
			s = strings.TrimSuffix(s, "b")
			s = strings.TrimSuffix(s, "k")
			s = strings.TrimSuffix(s, "m")
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				fsize = v
				setSize = true
			}
		}
	}

	fmt.Printf("fixBadDisk v%s\n", ver)
	fmt.Printf("Write: %v\n", doWrite)
	fmt.Printf("Read: %v\n", doTest)
	fmt.Printf("Path: %s\n", mustGetwd())

	if !(fileExists("bad") && fileExists("fixBadDiskWriteOK.txt")) {
		fmt.Println("Press Enter to run 按回车开始\n或者把需要测试的盘符拖进来按回车")
		var newPath string
		fmt.Scanln(&newPath)
		newPath = strings.TrimSpace(newPath)
		if newPath != "" {
			fmt.Printf("Path change to: %s\n", newPath)
			if err := os.Chdir(newPath); err != nil {
				fmt.Println("路径切换失败:", err)
				os.Exit(1)
			}
			doTest = fileExists("bad") && fileExists("fixBadDiskWriteOK.txt")
			doWrite = !fileExists("fixBadDiskWriteOK.txt")
			fmt.Printf("Write: %v\n", doWrite)
			fmt.Printf("Read: %v\n", doTest)
		}
	}

	if !fileExists("bad") {
		if err := os.Mkdir("bad", 0755); err != nil {
			fmt.Println("创建 bad 目录失败:", err)
			os.Exit(1)
		}
	}
	if err := os.Chdir("bad"); err != nil {
		fmt.Println("进入 bad 目录失败:", err)
		os.Exit(1)
	}
	badDir := mustGetwd()

	if !setSize {
		entries, _ := os.ReadDir(".")
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			nowSize := float64(info.Size()) / 1024 / 1024
			if math.Abs(nowSize-fsize) < 0.5 {
				break
			}
			fsize = nowSize
		}
		fmt.Printf("Filesize: %.3fM\n", fsize)
		if doWrite {
			entries2, _ := os.ReadDir(".")
			if len(entries2) == 0 {
				fmt.Println("Press Enter to run 按回车开始\n或输入自定义单文件大小(输入数字 单位MB)按回车")
				var input string
				fmt.Scanln(&input)
				input = strings.TrimSpace(input)
				if input != "" {
					if v, err := strconv.ParseFloat(input, 64); err == nil {
						fsize = v
						fmt.Printf("File size change to: %.0fM\n", fsize)
					}
				}
			}
		}
	}

	var saveSpeed []string
	var lastEcho string

	// ---- WRITE ----
	if doWrite {
		fmt.Print("\nWrite...\n\n")
		allt := 0.0
		minsp := math.MaxFloat64
		minavg := 0.0
		maxsp := 0.0

		var free float64
		if len(args) > 2 {
			if v, err := strconv.ParseFloat(args[2], 64); err == nil {
				free = v
			} else {
				free = getFreeSpaceMB(".")
			}
		} else {
			free = getFreeSpaceMB(".")
		}

		allCount := int(free / fsize)
		saveIndex := int(float64(allCount) * savePer)

		// 使用 worker pool 预生成文件，缓冲区大小为 CPU 核数
		gen := newFileGenerator(fsize, runtime.NumCPU())
		gen.start()

		for i := 0; i < allCount; i++ {
			result := <-gen.ch
			b := result.data
			n := result.name

			nt := 1e-10
			writeOK := false
			for !writeOK {
				st := time.Now()
				err := writeFileSync(n, b)
				nt = time.Since(st).Seconds()
				if err != nil {
					fmt.Printf("\nWrite Error %s\n%v\n", n, err)
					for connectErr(badDir) {
						fmt.Println("Connect Error 掉盘了！等待重连")
						time.Sleep(3 * time.Second)
					}
					_ = os.Remove(n)
					continue
				}
				allt += nt
				writeOK = true
			}

			if i == 0 {
				continue // 避免除零
			}

			ms := float64(i) * fsize / allt
			nsp := fsize / math.Max(nt, 1e-9)
			if nsp > maxsp {
				maxsp = nsp
			}
			if nsp < minsp {
				minsp = nsp
			}
			if nt > allt/float64(i) && (minavg == 0 || ms < minavg) {
				minavg = ms
			}
			remaining := float64(allCount-i) * fsize / ms
			lastEcho = fmt.Sprintf("Min: %.3fM/s Max: %.3fM/s Avg: %.3fM/s MinAvg: %.3fM/s\n%.3fM/%.3fM %s/%s (%.3fM/s %.6fs)",
				minsp, maxsp, ms, minavg,
				float64(i)*fsize, free,
				formatDuration(allt), formatDuration(remaining),
				nsp, nt,
			)
			fmt.Printf("\033[F\033[K%s\033[K  ", lastEcho)

			if i == saveIndex {
				per := (len(saveSpeed) + 1) * int(savePer*100)
				saveSpeed = append(saveSpeed, saveSpeedLine(per, minsp, maxsp, ms, minavg, nsp, nt))
				saveIndex = int(float64(len(saveSpeed)+1) * savePer * float64(allCount))
				fmt.Printf("\n%d%%\n\n", per)
				minsp = math.MaxFloat64
				maxsp = 0
				minavg = 0
			}
		}

		gen.stop()

		// 写入完成标识
		scoreContent := ""
		if len(lastEcho) > 6 {
			scoreContent = lastEcho[6:]
		}
		scoreContent += "\n" + strings.Join(saveSpeed, "\n")
		_ = writeFileSync("../fixBadDiskWriteOK.txt", []byte(scoreContent))

		// 填充剩余空间
		if len(args) <= 2 {
			remaining := getFreeSpaceMB(".")
			if remaining > 0 {
				data, ok := genFile(remaining)
				if ok {
					name := data2name(data)
					_ = writeFileSync(name, data)
				}
			}
		}

		fmt.Println("\n\nWrite complete, please unplug and reinsert the disk and run this program\n写入完成，请拔掉再插入磁盘并运行此程序")
	}

	// ---- READ / TEST ----
	var tIndex int64
	if doTest {
		writeScore := ""
		if fileExists("../fixBadDiskWriteOK.txt") {
			data, err := os.ReadFile("../fixBadDiskWriteOK.txt")
			if err == nil {
				writeScore = string(data)
				fmt.Printf("\nWrite Speed:\n\n%s\n", writeScore)
			} else {
				writeScore = "\nData Error! 扩容盘！\n" + "\nData Error! 扩容盘！\n" + "\nData Error! 扩容盘！\n"
				fmt.Print(writeScore)
			}
		}

		fmt.Print("\nTest...\n\n")
		allt := 0.0
		minsp := math.MaxFloat64
		minavg := 0.0
		maxsp := 0.0

		entries, err := os.ReadDir(".")
		if err != nil {
			fmt.Println("读取目录失败:", err)
			os.Exit(1)
		}
		// 只处理文件
		var files []string
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, e.Name())
			}
		}

		allCount := len(files)
		saveIndex := int(float64(allCount) * savePer)
		allsize := float64(allCount) * fsize
		saveSpeed = nil

		for i, key := range files {
			nt := 1e-10
			var d []byte
			readOK := false
			for !readOK {
				st := time.Now()
				d, err = os.ReadFile(key)
				nt = time.Since(st).Seconds()
				if err != nil {
					fmt.Printf("\nRead Error %s\n%v\n", key, err)
					for connectErr(badDir) {
						fmt.Println("Connect Error 掉盘了！等待重连")
						time.Sleep(3 * time.Second)
					}
					continue
				}
				allt += nt
				readOK = true
			}

			// 异步校验
			go func(data []byte, k string) {
				h := md5.Sum(data)
				name := hex.EncodeToString(h[:])[:8]
				if name == k {
					_ = os.Remove(k)
				} else {
					fmt.Printf("\nCheck Error %s\n", k)
				}
				atomic.AddInt64(&tIndex, 1)
			}(d, key)

			if i == 0 {
				continue
			}

			ms := float64(i) * fsize / allt
			nsp := fsize / math.Max(nt, 1e-9)
			if nsp > maxsp {
				maxsp = nsp
			}
			if nsp < minsp {
				minsp = nsp
			}
			if nt > allt/float64(i) && (minavg == 0 || ms < minavg) {
				minavg = ms
			}
			remaining := float64(allCount-i) * fsize / ms
			lastEcho = fmt.Sprintf("Min: %.3fM/s Max: %.3fM/s Avg: %.3fM/s MinAvg: %.3fM/s\n%.3fM/%.3fM %s/%s (%.3fM/s %.6fs)",
				minsp, maxsp, ms, minavg,
				float64(i)*fsize, allsize,
				formatDuration(allt), formatDuration(remaining),
				nsp, nt,
			)
			fmt.Printf("\033[F\033[K%s\033[K  ", lastEcho)

			if i >= saveIndex {
				per := (len(saveSpeed) + 1) * int(savePer*100)
				saveSpeed = append(saveSpeed, saveSpeedLine(per, minsp, maxsp, ms, minavg, nsp, nt))
				saveIndex = int(float64(len(saveSpeed)+1) * savePer * float64(allCount))
				fmt.Printf("\n%d%%\n\n", per)
				minsp = math.MaxFloat64
				maxsp = 0
				minavg = 0
			}
		}

		// 等待所有校验完成
		for atomic.LoadInt64(&tIndex) != int64(allCount) {
			time.Sleep(500 * time.Millisecond)
		}

		// 保存成绩
		scoreEcho := ""
		if len(lastEcho) > 6 {
			scoreEcho = lastEcho[6:]
		}
		_ = os.Remove("../fixBadDiskWriteOK.txt")
		f, err := os.OpenFile("../fixBadDiskScore.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			ts := time.Now().Format("2006-01-02 15:04:05")
			_, _ = fmt.Fprintf(f, "%s\r\nWrite Speed:\n%s\r\nRead Speed:\n%s\n%s\n\n",
				ts, writeScore, scoreEcho, strings.Join(saveSpeed, "\n"))
			_ = f.Close()
		}

		fmt.Println("\n\nTest complete 测试完成")
	}

	// 清理空 bad 目录
	_ = os.Chdir("..")
	if fileExists("bad") {
		entries, _ := os.ReadDir("bad")
		if len(entries) == 0 {
			_ = os.Remove("bad")
		}
	}

	fmt.Println("Press Enter to exit 按回车退出")
	fmt.Scanln()
}

// writeFileSync 写入文件并强制刷盘
func writeFileSync(name string, data []byte) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func mustGetwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.ToSlash(d)
}
