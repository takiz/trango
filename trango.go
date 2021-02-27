package main

//go:generate gotext -srclang=en update -out=catalog.go -lang=en,ru

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/famz/SetLocale"
	"github.com/gdamore/tcell/v2"
	"github.com/marksamman/bencode"
	"github.com/rivo/tview"
	"golang.org/x/sys/unix"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	_ "golang.org/x/text/message/catalog"
)

const (
	VERSION      = "0.1"
	DEFAULT_HOST = "127.0.0.1"
	DEFAULT_PORT = "9091"
	DEFAULT_URL  = "/transmission/rpc"
)

const (
	STATUS_STOPPED = iota
	STATUS_CHECK_WAIT
	STATUS_CHECK
	STATUS_DOWNLOAD_WAIT
	STATUS_DOWNLOAD
	STATUS_SEED_WAIT
	STATUS_SEED
	STATUS_ALL
	STATUS_ACTIVE
	STATUS_ERRORED
)

const (
	PRIORITY_LOW    = -1
	PRIORITY_NORMAL = 0
	PRIORITY_HIGH   = 1
	PRIORITY_MIXED  = 2
	WANTED          = 3
	UNWANTED        = 4
	WANTED_MIXED    = 5
	PRIORITY_SET    = 6
	WANTED_SET      = 7
)

const (
	MB       = 1048576
	GB       = 1024
	SPEED_KB = 1000
)

const (
	CATEGORY       = iota
	IN_GET_CURRENT // From UpdateCurrentTorrents()
	OUT_GET_CURRENT
	TORRENT_MOVE
	TORRENT_RENAME
	TRACKER_ADD
	TRACKER_RENAME
	KEYS // For SwitchToMain()
	LIST
	CONTENT
	TRACKERS
	ALL_T
	DIRS
	SAVE // To save a path/category.
)

// For printing hotkeys.
type Key struct {
	Name string
	Desc string
}

type StatusSymbol struct {
	Stopped   string
	CheckWait string
	Check     string
	DlWait    string
	Dl        string
	Seed      string
	Errored   string
}

// fmt.Sprintf("%*s") offset.
type StatFormat struct {
	Eta  int
	Done int
}

type Request struct {
	Method string      `json:"method"`
	Args   interface{} `json:"arguments"`
}

type Response struct {
	Args   interface{} `json:"arguments"`
	Result string      `json:"result"`
}

type Arg struct {
	Fields []string `json:"fields,omitempty"`
	Ids    []int    `json:"ids,omitempty"`
}

type Torrent struct {
	Desc     string
	Id       int      `json:"id,omitempty"`
	Name     string   `json:"name,omitempty"`
	Labels   []string `json:"labels,omitempty"`
	Status   int
	Path     string  `json:"downloadDir,omitempty"`
	Progress float64 `json:"percentDone,omitempty"`
	Size     int64   `json:"sizeWhenDone,omitempty"`
	Date     int     `json:"addedDate,omitempty"`
	DlSpeed  int     `json:"rateDownload,omitempty"`
	UplSpeed int     `json:"rateUpload,omitempty"`
	Error    int     `json:"error,omitempty"`
}

type TorrentInfo struct {
	DlSpeed  int     `json:"rateDownload,omitempty"`
	Eta      int64   `json:"eta,omitempty"`
	Id       int     `json:"id,omitempty"`
	Peers    int     `json:"peersConnected,omitempty"`
	Progress float64 `json:"percentDone,omitempty"`
	Size     int64   `json:"sizeWhenDone,omitempty"`
	Status   int     `json:"status,omitempty"`
	UplSpeed int     `json:"rateUpload,omitempty"`
	Error    int     `json:"error,omitempty"`
}

type GeneralInfo struct {
	Name         string   `json:"name,omitempty"`
	UploadRatio  float64  `json:"uploadRatio,omitempty"`
	UploadedEver int64    `json:"uploadedEver,omitempty"`
	HashString   string   `json:"hashString,omitempty"`
	DownloadDir  string   `json:"downloadDir,omitempty"`
	Comment      string   `json:"comment,omitempty"`
	Creator      string   `json:"creator,omitempty"`
	DateCreated  int64    `json:"dateCreated,omitempty"`
	AddedDate    int64    `json:"addedDate,omitempty"`
	TotalSize    int64    `json:"totalSize,omitempty"`
	ErrorString  string   `json:"errorString,omitempty"`
	Id           int      `json:"id,omitempty"`
	Labels       []string `json:"labels,omitempty"`
}

type SessionStats struct {
	ActiveTorrentCount int `json:"activeTorrentCount,omitempty"`
	PausedTorrentCount int `json:"pausedTorrentCount,omitempty"`
	TorrentCount       int `json:"torrentCount,omitempty"`
	DownloadSpeed      int `json:"downloadSpeed,omitempty"`
	UploadSpeed        int `json:"uploadSpeed,omitempty"`
}

type PeersInfo struct {
	Address       string  `json:"address,omitempty"`
	ClientName    string  `json:"clientName,omitempty"`
	FlagStr       string  `json:"flagStr,omitempty"`
	Progress      float64 `json:"progress,omitempty"`
	DownloadSpeed int     `json:"rateToClient,omitempty"`
	UploadSpeed   int     `json:"rateToPeer,omitempty"`
}

type TorrentsGet struct {
	All []*Torrent `json:"torrents"`
}

type TorrentsGetInfo struct {
	All []*TorrentInfo `json:"torrents"`
}

type Content struct {
	Name     string
	Desc     string
	Size     int64
	Id       int
	Progress float64
	Priority int
	DlFlag   int
	Path     string
	Dir      bool
	RootDir  bool
}

type Files struct {
	Name     string  `json:"name,omitempty"`
	Size     int64   `json:"length,omitempty"`
	Progress float64 `json:"bytesCompleted,omitempty"`
}

type FileStats struct {
	Priority int  `json:"priority,omitempty"`
	DlFlag   bool `json:"wanted,omitempty"`
}

type TorrentContent struct {
	Path      string      `json:"downloadDir,omitempty"`
	FileStats []FileStats `json:"fileStats,omitempty"`
	Files     []Files     `json:"files,omitempty"`
}

type TrackersInfo struct {
	Announce              string `json:"announce,omitempty"`
	Id                    int    `json:"id,omitempty"`
	LastAnnounceResult    string `json:"lastAnnounceResult,omitempty"`
	LastAnnouncePeerCount int    `json:"lastAnnouncePeerCount,omitempty"`
	SeederCount           int    `json:"seederCount,omitempty"`
}

type CurrStatus struct {
	Name string
	Id   int
}

// Tree item reference.
type Ref struct {
	Path   []string
	Name   string
	Dir    bool
	Id     int
	Length int64
}

type FileType struct {
	Dir    bool
	Length int64
	Id     int
}

type TreeFiles struct {
	Name      string
	FName     string
	Reference Ref
}

var (
	URL                      string
	HeaderId                 string
	AuthUsername, AuthPasswd string
	TransmissionVersion      int
	UpdateInt                time.Duration // Update info interval in seconds
	Torrents                 []*Torrent
	Stats                    *SessionStats
	Contents                 []Content
	ContentsTree             []Content
	FilePath                 string
	SelectedIds              map[int]int
	CurrentCategory          string
	CurrentStatus            CurrStatus
	Status                   map[string]int
	Category                 map[string]int
	Dirs                     map[string]int
	Title                    string
	TitleStatus              string
	MainKeysText             string
	SelectedFileIds          map[string]*FileType
	FilesAll                 []interface{}
	TotalSize                int64 // Current size of files in ShowAddDialog()
	StatSymb                 *StatusSymbol
	ALL, DEFAULT             string // Status/category names
	StatFmt                  = StatFormat{Eta: 6, Done: 7}
	RuneDir                  = string('\u23f7') + " "
	RuneLTee                 = " " + string(tcell.RuneLTee) + string(tcell.RuneHLine)
	RuneLTeeDir              = RuneLTee + RuneDir
	RuneLLCorner             = " " + string(tcell.RuneLLCorner) + string(tcell.RuneHLine) + " "
	RuneVLine                = "  " + string(tcell.RuneVLine)
	App                      *tview.Application
	Statusbar                *tview.TextView
	Header                   *tview.TextView
	Hotkeys                  *tview.TextView
	Peers                    *tview.TextView
	CategoryStatus           *tview.TextView
	SaveTo                   *tview.TextView
	CategoryName             *tview.TextView
	MainGrid                 *tview.Grid
	MainList                 *tview.List
	MainMutex                sync.Mutex
	PeersMutex               sync.Mutex
)

func SetOpts() {
	host := flag.String("host", DEFAULT_HOST, P("Set host"))
	port := flag.String("port", DEFAULT_PORT, P("Set port"))
	dir := flag.String("dir", "", P("<path>  Set download dir when adding a new torrent"))
	ctg := flag.String("category", "", P("<name1,name2,...>  Set categories when adding a new torrent"))
	filename := flag.String("add", "", P("<filename-or-URL>  Add torrent"))
	files := flag.String("files", "", P("<0,1,2,3,...> Mark files for download by index numbers"))
	user := flag.String("user", "", P("Set username"))
	pass := flag.String("pass", "", P("Set password"))
	ascii := flag.Bool("ascii", false, P("Show full status names"))
	start := flag.Bool("start", false, P("Start added torrent"))
	dialog := flag.Bool("dialog", false, P("Show dialog when adding a new torrent file (not url/magnet)"))
	trackers := flag.Bool("trackers", false, P("Print tracker URLs of a torrent file to standard output"))
	interval := flag.Int("update", 2, P("Set the interval for updating torrents information in seconds"))
	version := flag.Bool("version", false, P("Print current version"))

	flag.Parse()
	URL = "http://" + *host + ":" + *port + DEFAULT_URL
	if *user != "" || *pass != "" {
		u := make([]byte, len(*user))
		copy(u, *user)
		p := make([]byte, len(*pass))
		copy(p, *pass)
		AuthUsername = string(u)
		AuthPasswd = string(p)
		for j, ar := range os.Args[1:] {
			if ar == "-user" || ar == "-pass" {
				s := os.Args[j+2]
				max := len(s)
				for i := 0; i < max; i++ {
					h := (*reflect.StringHeader)(unsafe.Pointer(&s))
					a := (*int8)(unsafe.Pointer(h.Data + uintptr(i)))
					*a = '*'
				}
			}
		}
	}

	SelectedIds = make(map[int]int)
	UpdateInt = time.Duration(*interval)
	ALL = P("All")
	DEFAULT = P("Default")
	CurrentCategory = ALL
	CurrentStatus = CurrStatus{ALL, STATUS_ALL}
	Status = map[string]int{CurrentStatus.Name: 0}
	StatUni := StatusSymbol{
		fmt.Sprintf("[#D78700:]%s [-:]", string('\u25ae')),
		fmt.Sprintf("%s ", string('\u25cf')),
		fmt.Sprintf("%s ", string('\u2699')),
		fmt.Sprintf("%s ", string('\u29d7')),
		fmt.Sprintf("[blue:]%s [-:]", string('\U0001f81b')),
		fmt.Sprintf("[green:]%s [-:]", string('\U0001f819')),
		"[red:]! [-:]",
	}
	StatSymb = &StatUni
	if *ascii {
		pre := [...]string{P("Stopped"), P("Check wait"),
			P("Checking"), P("Queued"), P("Downloading"),
			P("Seeding"), P("Errored")}
		n := 0
		for _, s := range pre {
			l := utf8.RuneCountInString(s)
			if l > n {
				n = l
			}
		}
		StatAscii := StatusSymbol{
			fmt.Sprintf("[#D78700:]%-*s [-:]", n, pre[0]),
			fmt.Sprintf("%-*s ", n, pre[1]),
			fmt.Sprintf("%-*s ", n, pre[2]),
			fmt.Sprintf("%-*s ", n, pre[3]),
			fmt.Sprintf("[blue:]%-*s [-:]", n, pre[4]),
			fmt.Sprintf("[green:]%-*s [-:]", n, pre[5]),
			fmt.Sprintf("[red:]%-*s [-:]", n, pre[6]),
		}
		StatSymb = &StatAscii
		TitleStatus = fmt.Sprintf(" %-*s |", n, P("Status"))
	}
	l := utf8.RuneCountInString(P("Done"))
	if l > StatFmt.Done {
		StatFmt.Done = l
	}
	eta := P("ETA")
	l = utf8.RuneCountInString(eta)
	if l > StatFmt.Eta {
		StatFmt.Eta = l
	}
	Title = fmt.Sprintf("%*s ", StatFmt.Eta, eta) + P("|  Uploading  |"+
		" Downloading | Peers |  Done  |   Size    |") +
		TitleStatus + P("   Name ")

	MainKeysText = FormatKeys([]Key{{"F1", P("Help")}, {"F2", P("Status")},
		{"F3", P("Category")}, {"F4", P("General")}, {"F5", P("Trackers")},
		{"F6", P("Peers")}, {"F7", P("Search")}, {"F8", P("Content")},
		{"F9", P("Move")}, {"F10", P("Quit")}, {"F12", P("SortBy")}})

	if *version {
		fmt.Println(VERSION)
		os.Exit(0)
	}
	if *filename != "" {
		var notUrl bool
		if !strings.HasPrefix(*filename, "http") &&
			!strings.HasPrefix(*filename, "magnet") {
			notUrl = true
			if !strings.HasPrefix(*filename, "/") {
				pwd, err := os.Getwd()
				if err != nil {
					log.Fatal(err)
				}
				*filename = pwd + "/" + *filename
			}
		}
		if *trackers {
			_, _, _, tr := ParseTorrent(filename)
			for _, t := range tr {
				fmt.Fprintf(os.Stdout, "%s\n", t)
			}
			os.Exit(0)
		}
		var cancelDlg bool
		if *dialog && notUrl {
			ShowAddDialog(filename, files, ctg, dir, start, &cancelDlg)
		}
		paused := true
		if *start {
			paused = false
		}
		if !cancelDlg {
			AddTorrent(*filename, *dir, *ctg, *files, paused)
		}
		os.Exit(0)
	}
}

func main() {
	SetLocales()
	SetOpts()
	GetSessionStats()
	TransmissionVersion = GetVersion()
	GetTorrents()

	CategoryStatus = NewTextPrim(PrintCtgStat())
	Header = NewTextPrim(Title)
	Header.SetBorder(true).SetBorderColor(tcell.ColorDefault)
	Statusbar = NewTextPrim(" ")
	Statusbar.SetBorder(true).SetBorderColor(tcell.ColorDefault)
	Hotkeys = NewTextPrim(MainKeysText)

	InitMainList()

	MainGrid = tview.NewGrid().
		SetRows(1, 3, 0, 3, 1).
		SetColumns(30, 30, 0).
		SetBorders(false).
		AddItem(CategoryStatus, 0, 0, 1, 3, 0, 0, false).
		AddItem(Header, 1, 0, 1, 3, 0, 0, false).
		AddItem(MainList, 2, 0, 1, 3, 0, 0, true).
		AddItem(Statusbar, 3, 0, 1, 3, 0, 0, false).
		AddItem(Hotkeys, 4, 0, 1, 3, 0, 0, false)

	MainGrid.SetBackgroundColor(tcell.ColorDefault)
	App = tview.NewApplication().SetRoot(MainGrid, true)
	App.SetBeforeDrawFunc(func(s tcell.Screen) bool {
		s.Clear()
		return false
	})
	SetMainInput()
	go ShowCurrent()
	if err := App.Run(); err != nil {
		panic(err)
	}
}

func SetMainInput() {
	SetKeysHeaderText(Title, MainKeysText, tview.AlignLeft)
	App.SetFocus(MainList).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyF1:
				ShowHelpInfo()
			case tcell.KeyF2:
				ShowStatusInfo()
			case tcell.KeyF3:
				if TransmissionVersion < 3 {
					ShowVersionInfo(MainList, LIST, App.GetInputCapture())
				} else {
					ShowCategoryInfo()
				}
			case tcell.KeyF4:
				ShowGeneralInfo(MainList.GetCurrentItem())
			case tcell.KeyF5:
				MainMutex.Lock()
				ShowTrackersInfo(MainList, 0)
			case tcell.KeyF6:
				ShowPeersInfo(MainList.GetCurrentItem())
			case tcell.KeyF7:
				input := App.GetInputCapture()
				ShowSearchInput(MainList, KEYS, input)
			case tcell.KeyF8:
				ShowContentInfo(MainList.GetCurrentItem())
			case tcell.KeyF9:
				ShowInputField(MainList, TORRENT_MOVE, nil)
			case tcell.KeyF10:
				App.Stop()
			case tcell.KeyF12:
				SortTorrents()
			case tcell.KeyCtrlL:
				ShowInputField(MainList, TORRENT_RENAME, nil)
			case tcell.KeyCtrlP:
				TorAction(MainList.GetCurrentItem(), "torrent-stop", true)
			case tcell.KeyCtrlS:
				TorAction(MainList.GetCurrentItem(), "torrent-start", true)
			case tcell.KeyCtrlR:
				TorAction(MainList.GetCurrentItem(), "torrent-verify", true)
			case tcell.KeyCtrlF:
				TorAction(MainList.GetCurrentItem(), "torrent-reannounce", true)
			case tcell.KeyDelete:
				ShowConfirmation("torrent(s)",
					"torrent-remove", false)
			case tcell.KeyCtrlA:
				SelectAll(MainList, true)
			case tcell.KeyEsc:
				SelectAll(MainList, false)
			case tcell.KeyCtrlN:
				if TransmissionVersion < 3 {
					ShowVersionInfo(MainList, LIST, App.GetInputCapture())
				} else {
					ShowInputField(MainList, CATEGORY, nil)
				}
			case tcell.KeyCtrlU:
				OpenAction("comment")
			case tcell.KeyCtrlO:
				OpenAction("downloadDir")
			case tcell.KeyEnter:
				id := GetId(MainList.GetCurrentItem(), MainList)
				PreviewFile(id)
			case tcell.KeyRune:
				switch event.Rune() {
				case ' ': // space
					SelectItem(MainList)
				case '~': // ~ or shift+delete
					ShowConfirmation("torrent(s)",
						"torrent-remove", true)
				}
			}
			return event
		})
}

func ShowVersionInfo(p tview.Primitive, r int, input func(event *tcell.EventKey) *tcell.EventKey) {
	modal := tview.NewModal().
		SetText(P("You need transmission-daemon version 3.00 or later for the categories support.")).
		AddButtons([]string{"OK"})
	if r == LIST { // From Main()
		MainGrid.AddItem(modal, 2, 0, 1, 3, 0, 0, true)
	} else { // From Add Dialog.
		MainGrid.AddItem(modal, 5, 0, 1, 5, 0, 0, true)
	}
	App.SetFocus(modal).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc, tcell.KeyEnter:
				MainGrid.RemoveItem(modal)
				App.SetFocus(p).SetInputCapture(input)
			}
			return event
		})
}

func NewInputFieldPrim(label string) *tview.InputField {
	inp := tview.NewInputField().
		SetLabel(label).
		SetFieldWidth(50).SetFieldTextColor(tcell.Color221).
		SetFieldBackgroundColor(tcell.Color25).
		SetLabelColor(tcell.ColorDefault)
	inp.SetBackgroundColor(tcell.ColorDefault)
	return inp
}

func NewTextPrim(text string) *tview.TextView {
	t := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText(text).SetTextColor(tcell.ColorDefault)
	t.SetBackgroundColor(tcell.ColorDefault)
	return t
}

func NewListPrim() *tview.List {
	l := tview.NewList().ShowSecondaryText(false).
		SetSelectedBackgroundColor(tcell.Color25).
		SetSelectedTextColor(tcell.Color221).
		SetMainTextColor(tcell.ColorDefault).
		SetHighlightFullLine(true)
	l.SetBackgroundColor(tcell.ColorDefault)
	return l
}

func SortTorrents() {
	MainMutex.Lock()
	MainGrid.RemoveItem(MainList)
	keys := []Key{{"Esc", P("Close")}, {"Enter", P("Sort")}}
	SetKeysHeaderText(P("Sort by"), FormatKeys(keys), tview.AlignCenter)
	list := NewListPrim().
		AddItem(P("   Added Date"), "", 0, nil).
		AddItem(P("   Name"), "", 0, nil).
		AddItem(P("   Progress"), "", 0, nil).
		AddItem(P("   Size"), "", 0, nil)

	endwin := func() {
		list.Clear()
		SwitchToMain(list, LIST)
		MainMutex.Unlock()
	}
	MainGrid.AddItem(list, 2, 0, 1, 3, 0, 0, true)
	App.SetFocus(list).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				endwin()
			case tcell.KeyEnter:
				item := list.GetCurrentItem()
				switch item {
				case 0:
					sort.Slice(Torrents, func(i, j int) bool {
						return Torrents[i].Date > Torrents[j].Date
					})
				case 1:
					sort.Slice(Torrents, func(i, j int) bool {
						return strings.ToLower(Torrents[i].Name) < strings.ToLower(Torrents[j].Name)
					})
				case 2:
					sort.Slice(Torrents, func(i, j int) bool {
						return Torrents[i].Progress > Torrents[j].Progress
					})
				case 3:
					sort.Slice(Torrents, func(i, j int) bool {
						return Torrents[i].Size > Torrents[j].Size
					})
				}
				MainList.Clear()
				InitMainList()
				endwin()
			}
			return event
		})
}

func DiskAvail(path string) string {
	avText := ""
	fs := unix.Statfs_t{}
	if err := unix.Statfs(path, &fs); err == nil {
		a := int64(fs.Bavail) * int64(fs.Bsize)
		avText = fmt.Sprintf(" ([::u]%s "+P("Free")+"[::-])", FormatSize(a))
	}
	return avText
}

// Input field when creating a new path/category.
func ShowInputCtgPath(list *tview.List, tree *tview.TreeView, r int, mainKeys, mainHeader string, ctg, dir *string, mainInput, input func(event *tcell.EventKey) *tcell.EventKey) {
	var s, t string
	MainGrid.RemoveItem(Hotkeys)
	item := list.GetCurrentItem()
	if r == CATEGORY {
		s = P("Enter a new category name(s):")
	} else { //DIRS
		s = P("Enter a new path:")
		t, _ = list.GetItemText(item)
	}
	keys := []Key{{"Esc", P("Cancel")}}
	inputField := NewInputFieldPrim(FormatKeys(keys) + s).SetText(t)
	MainGrid.AddItem(inputField, 6, 0, 1, 4, 0, 0, false)
	endwin := func() {
		MainGrid.RemoveItem(inputField)
		list.Clear()
		MainGrid.RemoveItem(list)
		MainGrid.AddItem(tree, 5, 0, 1, 5, 0, 0, true)
		MainGrid.AddItem(Hotkeys, 6, 0, 1, 5, 0, 0, false)
		Hotkeys.SetText(mainKeys)
		Header.SetText(mainHeader)
		App.SetFocus(tree).SetInputCapture(mainInput)
	}
	App.SetFocus(inputField).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				MainGrid.RemoveItem(inputField)
				MainGrid.AddItem(Hotkeys, 6, 0, 1, 4, 0, 0, false)
				App.SetFocus(list).SetInputCapture(input)
			case tcell.KeyEnter:
				s := inputField.GetText()
				tLen := len(s)
				if r == DIRS && tLen > 0 {
					*dir = s
					avText := DiskAvail(*dir)
					SaveTo.SetText(P(" Path") + avText + ": " + s)
				} else if tLen > 0 {
					if s == DEFAULT {
						*ctg = ""
					} else {
						*ctg = s
					}
					CategoryName.SetText(P(" Category: ") + s)
				}
				endwin()
			}
			return event
		})
}

// To select path/category in the Add Dialog.
func AddDialogShowCtgDirs(r int, tree *tview.TreeView, ctg, dir *string, input func(event *tcell.EventKey) *tcell.EventKey) {
	MainGrid.RemoveItem(tree)
	mainKeys := Hotkeys.GetText(false)
	mainHeader := Header.GetText(false)
	PrintKeys(r)
	list := NewListPrim()
	if r == DIRS {
		Dirs = make(map[string]int)
		for _, t := range Torrents {
			Dirs[strings.TrimSuffix(t.Path, "/")]++
		}
		d := make([]string, 0)
		for k, _ := range Dirs {
			d = append(d, k)
		}
		sort.Strings(d)
		for _, k := range d {
			list.AddItem(k, k, 0, nil)
		}
	} else {
		InitCategory(CATEGORY, list)
	}
	endwin := func() {
		list.Clear()
		MainGrid.RemoveItem(list)
		MainGrid.AddItem(tree, 5, 0, 1, 5, 0, 0, true)
		Hotkeys.SetText(mainKeys)
		Header.SetText(mainHeader)
		App.SetFocus(tree).SetInputCapture(input)
	}
	MainGrid.AddItem(list, 5, 0, 1, 5, 0, 0, true)
	App.SetFocus(list).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				endwin()
			case tcell.KeyF2:
				listInput := App.GetInputCapture()
				ShowInputCtgPath(list, tree, r, mainKeys, mainHeader,
					ctg, dir, input, listInput)
			case tcell.KeyF7:
				ShowSearchInput(list, r, App.GetInputCapture())
			case tcell.KeyEnter:
				item := list.GetCurrentItem()
				_, s := list.GetItemText(item)
				if r == DIRS {
					*dir = s
					avText := DiskAvail(*dir)
					SaveTo.SetText(P(" Path") + avText + ": " + s)
				} else {
					if s == DEFAULT {
						*ctg = ""
					} else {
						*ctg = s
					}
					CategoryName.SetText(P(" Category: ") + s)
				}
				endwin()
			}
			return event
		})
}

// Save/restore the last path/category from the file.
func SetLast(r int, dir, ctg *string) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	d := configDir + "/trango/"
	if err := os.MkdirAll(d, 0774); err != nil {
		log.Fatal(err)
	}
	var f *os.File
	if r == SAVE {
		f, err = os.OpenFile(d+"last", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	} else {
		f, err = os.OpenFile(d+"last", os.O_RDONLY|os.O_CREATE, 0644)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if r == SAVE {
		c := *ctg
		if *ctg == DEFAULT || *ctg == "" {
			c = "Default"
		}
		fmt.Fprintf(f, "%s\n%s\n", *dir, c)
		return
	}
	sc := bufio.NewScanner(f)
	i := 0
	for ; sc.Scan(); i++ {
		t := sc.Text()
		if i == 0 {
			*dir = t
		} else {
			if t == "Default" {
				*ctg = DEFAULT
			} else {
				*ctg = t
			}
		}
	}
	if i != 1 {
		if *dir == "" {
			type SessionSettings struct {
				DownloadDir string `json:"download-dir,omitempty"`
			}
			in := &Request{
				Method: "session-get",
			}
			out := &Response{Args: &SessionSettings{}}
			GetRequest(in, out)
			*dir = out.Args.(*SessionSettings).DownloadDir
		}
		c := *ctg
		if *ctg == "" {
			c = "Default"
			*ctg = DEFAULT
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		f, err = os.OpenFile(d+"last", os.O_WRONLY|os.O_TRUNC, 0644)
		fmt.Fprintf(f, "%s\n%s\n", *dir, c)
	}
}

func ShowAddDialog(filename, files, ctg, dir *string, start, cancel *bool) {
	var rootDir []string
	var rootLength int64
	SelectedFileIds = make(map[string]*FileType)
	rootDir, FilesAll, rootLength, _ = ParseTorrent(filename)
	nFiles := len(FilesAll)
	TransmissionVersion = GetVersion()
	GetSessionStats()
	GetCtgDirs()
	root := tview.NewTreeNode(rootDir[0])
	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	length := TreeAdd(root, rootDir, true)
	if nFiles == 0 {
		length = rootLength
	}
	root.Walk(SelectTreeItem)
	root.SetText(fmt.Sprintf("%s (%s)", rootDir[0], FormatSize(length)))
	TotalSize = length

	TreeSelected := func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference == nil {
			return // Selecting the root node does nothing.
		}
		children := node.GetChildren()
		if len(children) == 0 {
			path := reference.(Ref).Path
			TreeAdd(node, path, false)
		} else {
			// Collapse if visible, expand if collapsed.
			node.SetExpanded(!node.IsExpanded())
		}
	}

	Header = NewTextPrim(P("Add torrent")).SetTextAlign(tview.AlignCenter)
	Header.SetBorder(true).SetBorderColor(tcell.ColorDefault)
	if *dir == "" {
		SetLast(DIRS, dir, ctg)
	}
	avText := DiskAvail(*dir)
	SaveTo = NewTextPrim(P(" Path") + avText + ": " + *dir)
	if *ctg == "" {
		*ctg = DEFAULT
		SetLast(CATEGORY, dir, ctg)
	}
	CategoryName = NewTextPrim(P(" Category: ") + *ctg)
	startText := P(" Start torrent:") + " " + P("no")
	if *start {
		startText = P(" Start torrent:") + " " + P("yes")
	}
	StartTorrent := NewTextPrim(startText)
	CurrentSize := NewTextPrim(P(" Size") + ": " + FormatSize(TotalSize))
	keysText := []Key{{P("Space"), P("Get")}, {"Enter", P("(Un)expand dir")},
		{"Esc", P("Cancel")}, {"F1", "OK"}, {"F2", P("Start yes/no")},
		{"F3", P("Category")}, {"F4", P("Path")}}
	Hotkeys = NewTextPrim(FormatKeys(keysText))
	MainGrid = tview.NewGrid().
		SetRows(3, 1, 1, 1, 2, 0, 1).
		SetColumns(30, 30, 30, 30, 30, 0).
		SetBorders(false).
		AddItem(Header, 0, 0, 1, 5, 0, 0, false).
		AddItem(SaveTo, 1, 0, 1, 5, 0, 0, false).
		AddItem(CategoryName, 2, 0, 1, 5, 0, 0, false).
		AddItem(StartTorrent, 3, 0, 1, 5, 0, 0, false).
		AddItem(CurrentSize, 4, 0, 1, 5, 0, 0, false).
		AddItem(tree, 5, 0, 1, 5, 0, 0, true).
		AddItem(Hotkeys, 6, 0, 1, 5, 0, 0, false)

	MainGrid.SetBackgroundColor(tcell.ColorDefault)
	App = tview.NewApplication().SetRoot(MainGrid, true)
	App.SetBeforeDrawFunc(func(s tcell.Screen) bool {
		s.Clear()
		return false
	})
	App.SetFocus(tree).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				*cancel = true
				if *ctg == DEFAULT {
					*ctg = ""
				}
				App.Stop()
			case tcell.KeyF1:
				if *files != "" {
					*files += ","
				}
				for _, val := range SelectedFileIds {
					if !val.Dir {
						*files += strconv.Itoa(val.Id) + ","
					}
				}
				if nFiles == 0 {
					*files = "0"
				}
				SetLast(SAVE, dir, ctg)
				if *ctg == DEFAULT {
					*ctg = ""
				}
				App.Stop()
			case tcell.KeyF2:
				if *start {
					*start = false
					StartTorrent.SetText(P(" Start torrent:") + " " + P("no"))
				} else {
					*start = true
					StartTorrent.SetText(P(" Start torrent:") + " " + P("yes"))
				}
			case tcell.KeyF3:
				input := App.GetInputCapture()
				if TransmissionVersion < 3 {
					ShowVersionInfo(tree, CATEGORY, input)
				} else {
					AddDialogShowCtgDirs(CATEGORY, tree, ctg, dir, input)
				}
			case tcell.KeyF4:
				input := App.GetInputCapture()
				AddDialogShowCtgDirs(DIRS, tree, ctg, dir, input)
			case tcell.KeyEnter:
				node := tree.GetCurrentNode()
				r := node.GetReference()
				if r != nil && r.(Ref).Dir {
					TreeSelected(node)
				}
			case tcell.KeyRune:
				switch event.Rune() {
				case ' ':
					node := tree.GetCurrentNode()
					node.Walk(SelectTreeItem)
					CurrentSize.SetText(P(" Size") + ": " + FormatSize(TotalSize))
				}
			}
			return event
		})

	if err := App.Run(); err != nil {
		panic(err)
	}
}

func TreeRestoreExpandSelected(node *tview.TreeNode, flag bool) {
	r := node.GetReference()
	if r == nil {
		return
	}
	if flag && r.(Ref).Dir {
		node.SetExpanded(false)
	} else if r.(Ref).Dir {
		node.SetExpanded(true)
	}
}

// Add files to the tree (Add Dialog).
func TreeAdd(target *tview.TreeNode, path []string, root bool) int64 {
	files, length := ReadFiles(FilesAll, path, root)
	filesSrt := make([]TreeFiles, len(files))
	i := 0
	for file, t := range files {
		p := Ref{Name: file, Id: t.Id}
		if !root {
			p.Path = append(p.Path, path...)
		}
		p.Path = append(p.Path, file)
		if t.Dir {
			filesSrt[i].FName = fmt.Sprintf("[ ]%s", file)
			p.Dir = true
		} else {
			filesSrt[i].FName = fmt.Sprintf("[ ]%s (%s)",
				file, FormatSize(t.Length))
			p.Length = t.Length
		}
		filesSrt[i].Name = file
		filesSrt[i].Reference = p
		i++
	}
	sort.Slice(filesSrt, func(i, j int) bool {
		return filesSrt[i].Name < filesSrt[j].Name
	})
	for _, f := range filesSrt {
		node := tview.NewTreeNode(f.FName).SetReference(f.Reference)
		if f.Reference.Dir {
			node.SetColor(tcell.ColorRed)
		}
		target.AddChild(node)
	}
	return length
}

// Read files from torrent (when adding).
func ReadFiles(files []interface{}, path []string, root bool) (map[string]*FileType, int64) {
	pathEq := func(p []interface{}, s []string) bool {
		count := 0
		max := len(p)
		for i, t := range s {
			if i < max && t == p[i] {
				count++
			}
		}
		if count == len(s) {
			return true
		}
		return false
	}
	var length int64
	filenames := make(map[string]*FileType)
	max := len(path)
	index := 0
	for _, v2 := range files {
		le1 := v2.(map[string]interface{})["length"]
		p := v2.(map[string]interface{})["path"].([]interface{})
		for k3, v3 := range v2.(map[string]interface{}) {
			if k3 == "path" {
				f := v3.([]interface{})
				if root {
					filenames[p[0].(string)] = &FileType{Id: index}
					filenames[p[0].(string)].Length = le1.(int64)
					if len(f) > 1 {
						filenames[p[0].(string)].Dir = true
					}
					length += le1.(int64)
				} else if pathEq(f, path) {
					filenames[p[max].(string)] = &FileType{Id: index}
					filenames[p[max].(string)].Length = le1.(int64)
					if len(f) > max+1 {
						filenames[p[max].(string)].Dir = true
					}
					length += le1.(int64)
				}
			}
		}
		index++
	}
	// fmt.Println(FormatSize(length), path[max-1])
	return filenames, length
}

func SelectTreeItem(node, parent *tview.TreeNode) bool {
	r := node.GetReference()
	if r == nil {
		return true // Selecting the root node does nothing.
	}
	var length int64
	if r.(Ref).Dir {
		children := node.GetChildren()
		if len(children) == 0 {
			path := r.(Ref).Path
			length = TreeAdd(node, path, false)
		}
	}
	id := r.(Ref).Id
	name := fmt.Sprintf("%s_%d", r.(Ref).Name, id)
	if _, ok := SelectedFileIds[name]; ok {
		delete(SelectedFileIds, name)
		text := node.GetText()
		if r.(Ref).Dir && length != 0 {
			node.SetText(fmt.Sprintf("[ ]%s (%s)", text[3:],
				FormatSize(length)))
		} else {
			node.SetText(fmt.Sprintf("[ ]%s", text[3:]))
		}
		TotalSize -= r.(Ref).Length
	} else {
		SelectedFileIds[name] = &FileType{Id: id, Dir: r.(Ref).Dir}
		text := node.GetText()
		if r.(Ref).Dir && length != 0 {
			node.SetText(fmt.Sprintf("[*]%s (%s)", text[3:],
				FormatSize(length)))
		} else {
			node.SetText(fmt.Sprintf("[*]%s", text[3:]))
		}
		TotalSize += r.(Ref).Length
	}
	return true
}

func ParseTorrent(filename *string) ([]string, []interface{}, int64, []string) {
	file, err := os.Open(*filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	dict, err := bencode.Decode(file)
	if err != nil {
		log.Fatal(err)
	}
	var length int64
	name := make([]string, 0)
	trackers := make([]string, 0)
	var files []interface{}
	for key, value := range dict {
		switch key {
		case "info":
			for k, v := range value.(map[string]interface{}) {
				switch k {
				case "name":
					name = append(name, v.(string))
				case "length":
					length = v.(int64)
				case "files":
					files = v.([]interface{})
				}
			}
		case "announce":
			trackers = append(trackers, value.(string))
		case "announce-list":
			for _, v2 := range value.([]interface{}) {
				trackers = append(trackers, v2.([]interface{})[0].(string))
			}
		}
	}
	return name, files, length, trackers
}

func AddTorrent(filename, dir, ctg, files string, paused bool) {
	type arg struct {
		Filename    string `json:"filename"`
		Paused      bool   `json:"paused"`
		DownloadDir string `json:"download-dir,omitempty"`
		// Labels      []string `json:"labels"`
	}
	in := &Request{
		Args: arg{
			Filename:    filename,
			Paused:      paused,
			DownloadDir: dir,
			// Labels:      []string{ctg},
		},
		Method: "torrent-add",
	}
	type tor struct {
		Name       string `json:"name"`
		Id         int    `json:"id"`
		HashString string `json:"hashString"`
	}
	type rArg struct {
		Duplicate tor `json:"torrent-duplicate,omitempty"`
		Added     tor `json:"torrent-added,omitempty"`
	}
	out := &Response{Args: &rArg{}}
	GetRequest(in, out)
	res := out.Args.(*rArg).Duplicate
	if res.HashString != "" {
		fmt.Fprintln(os.Stderr, P("Torrent already added"))
	} else {
		res = out.Args.(*rArg).Added
	}

	if ctg != "" {
		id := res.Id
		labels := strings.Split(ctg, ",")
		type arg struct {
			Labels []string `json:"labels"`
			Ids    []int    `json:"ids"`
		}
		in = &Request{
			Args: arg{
				Labels: labels,
				Ids:    []int{id},
			},
			Method: "torrent-set",
		}
		out = &Response{}
		GetRequest(in, out)
	}
	if files != "" {
		id := res.Id
		in = &Request{}
		out = &Response{}
		pdata := []byte(fmt.Sprintf(`{"method":"torrent-set","arguments":{"files-unwanted": [], "files-wanted": [ %s ],"ids":[%d]}}`, strings.TrimSuffix(files, ","), id))
		GetRequest(in, out, pdata)
	}
}

func ShowHelpInfo() {
	MainMutex.Lock()
	MainGrid.RemoveItem(MainList)
	keys := []Key{{"Esc", P("Close")}}
	SetKeysHeaderText(P("Hotkeys"), FormatKeys(keys), tview.AlignCenter)
	text := fmt.Sprintf(" [red:]Ctrl+S[-:-]: " + P("start") + "\n" +
		" [red:]Ctrl+P[-:-]: " + P("stop") + "\n" +
		" [red:]Ctrl+R[-:-]: " + P("verify") + "\n" +
		" [red:]Ctrl+F[-:-]: " + P("reannounce") + "\n" +
		" [red:]Delete[-:-]: " + P("remove torrent(s)") + "\n" +
		" [red:]Shift+Delete[-:-] " + P("or") + " [red:]~[-:-]: " + P("remove torrent(s) with data") + "\n" +
		" [red:]Enter[-:-]: " + P("preview/open file(s)") + "\n" +
		" [red:]Space[-:-]: " + P("select/unselect") + "\n" +
		" [red:]Ctrl+A[-:-]: " + P("select all") + "\n" +
		" [red:]Esc[-:-]: " + P("cancel selection") + "\n" +
		" [red:]Ctrl+N[-:-]: " + P("create a new category for selected torrent(s)") + "\n" +
		" [red:]Ctrl+U[-:-]: " + P("open comment url") + "\n" +
		" [red:]Ctrl+O[-:-]: " + P("open download dir") + "\n" +
		" [red:]Ctrl+L[-:-]: " + P("rename torrent") + "\n")
	hi := NewTextPrim(text)
	MainGrid.AddItem(hi, 2, 0, 1, 3, 0, 0, true)
	App.SetFocus(hi).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				hi.Clear()
				SwitchToMain(hi, LIST)
				MainMutex.Unlock()
			}
			return event
		})
}

func ShowContentPreview() {
	MainMutex.Lock()
	cpr := NewListPrim()
	count := 0
	for _, t := range Contents {
		if int64(t.Progress) == t.Size {
			var name string
			end := strings.LastIndex(t.Name, "/")
			if end == -1 {
				name = t.Name
			} else {
				name = t.Name[end+1:]
			}
			pr := 0.0
			if t.Size > 0 {
				pr = t.Progress / float64(t.Size)
			}
			cpr.AddItem(fmt.Sprintf("  %*s  %8s   %s",
				StatFmt.Done, FormatProgress(pr),
				FormatSize(t.Size), name),
				fmt.Sprintf("%s", t.Name), 0, nil)
			count++
		}
	}
	if count == 0 {
		cpr.Clear()
		MainMutex.Unlock()
		return
	}
	MainGrid.RemoveItem(MainList)
	title := P("  Done  |  Size   |  Name ")
	Header.SetTextAlign(tview.AlignLeft).SetText(title)
	keys := []Key{{"Esc", P("Close")}, {"Enter", P("Open")}}
	Hotkeys.SetText(FormatKeys(keys))
	MainGrid.AddItem(cpr, 2, 0, 1, 3, 0, 0, true)
	App.SetFocus(cpr).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				Contents = nil
				FilePath = ""
				cpr.Clear()
				SwitchToMain(cpr, LIST)
				MainMutex.Unlock()
			case tcell.KeyEnter:
				item := cpr.GetCurrentItem()
				_, s := cpr.GetItemText(item)
				OpenItem(FilePath + "/" + s)
			}
			return event
		})
}

func PreviewFile(id int) {
	GetContentInfo(id)
	n := len(Contents)
	if n == 0 {
		return
	} else if n == 1 && int64(Contents[0].Progress) == Contents[0].Size {
		OpenItem(FilePath + "/" + Contents[0].Name)
		Contents = nil
		FilePath = ""
	} else if n > 1 {
		ShowContentPreview()
	}
}

func OpenItem(s string) {
	cmd := exec.Command("xdg-open", s)
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
}

func OpenAction(r string) {
	item := MainList.GetCurrentItem()
	id := GetId(item, MainList)
	res := GetAction(id, r)
	if len(res) > 0 {
		OpenItem(res)
	}
}

func TrackerAction(id, trackerId int, s, argName string) {
	var q string
	in := &Request{}
	out := &Response{}
	if argName == "trackerReplace" {
		q = (fmt.Sprintf("[%d, %q]", trackerId, s))
	} else if argName == "trackerAdd" {
		q = (fmt.Sprintf("[ %q ]", s))
	} else { //trackerRemove
		q = (fmt.Sprintf("[ %d ]", trackerId))
	}
	pdata := []byte(fmt.Sprintf(`{"method":"torrent-set","arguments":{%q:%s,"ids":[%d]}}`, argName, q, id))
	GetRequest(in, out, pdata)
}

func MovieTorrent(id int, dir string) {
	var ids []int
	for i, _ := range SelectedIds {
		ids = append(ids, i)
	}
	if len(ids) == 0 {
		ids = append(ids, id)
	}
	type arg struct {
		Location string `json:"location"`
		Move     bool   `json:"move"`
		Ids      []int  `json:"ids"`
	}
	in := &Request{
		Args: arg{
			Location: dir,
			Move:     true,
			Ids:      ids,
		},
		Method: "torrent-set-location",
	}
	out := &Response{}
	GetRequest(in, out)
}

func RenameTorrent(id int, name, newName string) bool {
	type arg struct {
		Path string `json:"path"`
		Name string `json:"name"`
		Ids  []int  `json:"ids"`
	}
	in := &Request{
		Args: arg{
			Path: name,
			Name: newName,
			Ids:  []int{id},
		},
		Method: "torrent-rename-path",
	}
	out := &Response{}
	GetRequest(in, out)
	if out.Result == "success" {
		return true
	}
	return false
}

func ShowConfirmation(s, method string, flag bool) {
	keys := []Key{{"Esc", P("No")}, {"Enter", P("Yes")}}
	Hotkeys.SetText(fmt.Sprintf("%s [white:red] "+P("Do you really want to delete")+" %s?", FormatKeys(keys), s))
	App.SetFocus(Hotkeys).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				SwitchToMain(Hotkeys, KEYS)
			case tcell.KeyEnter:
				MainMutex.Lock()
				TorAction(MainList.GetCurrentItem(), method, flag)
				SelectedIds = make(map[int]int)
				MainMutex.Unlock()
				SwitchToMain(Hotkeys, KEYS)
			}
			return event
		})
}

func ShowInputField(list *tview.List, r int, input func(event *tcell.EventKey) *tcell.EventKey) {
	var s, t, dir string
	var id, trackerId int
	MainGrid.RemoveItem(Hotkeys)
	item := list.GetCurrentItem()
	switch r {
	case CATEGORY:
		s = P("Enter a new category name(s):")
	case TORRENT_MOVE:
		s = P("Move to:")
		id = GetId(item, list)
		dir = GetAction(id, "downloadDir")
		t = dir
	case TORRENT_RENAME:
		s = P("Rename to:")
		id = GetId(item, list)
		for _, tor := range Torrents {
			if tor.Id == id {
				t = tor.Name
				break
			}
		}
	case TRACKER_ADD:
		s = P("Enter announce URL:")
		id = GetId(MainList.GetCurrentItem(), MainList)
	case TRACKER_RENAME:
		s = P("Tracker URL:")
		id = GetId(MainList.GetCurrentItem(), MainList)
		trackerId = GetId(item, list)
		m, _ := list.GetItemText(item)
		for i, b := range m {
			if unicode.IsSpace(b) {
				t = m[:i]
				break
			}
		}
	}
	keys := []Key{{"Esc", P("Cancel")}}
	inputField := NewInputFieldPrim(FormatKeys(keys) + s).SetText(t)
	MainGrid.AddItem(inputField, 4, 0, 1, 3, 0, 0, false)
	updateList := func() {
		MainGrid.RemoveItem(inputField)
		MainGrid.RemoveItem(list)
		curItem := list.GetCurrentItem()
		list.Clear()
		MainGrid.AddItem(Hotkeys, 4, 0, 1, 3, 0, 0, false)
		ShowTrackersInfo(list, curItem)
	}
	App.SetFocus(inputField).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				switch r {
				case CATEGORY, TORRENT_MOVE, TORRENT_RENAME:
					SwitchToMain(inputField, KEYS)
				case TRACKER_ADD, TRACKER_RENAME:
					SetPrevInput(inputField, list, TRACKERS, input)
				}
			case tcell.KeyEnter:
				text := inputField.GetText()
				tLen := len(text)
				switch r {
				case CATEGORY:
					MainMutex.Lock()
					res := SetNewCategory(list.GetCurrentItem(), text)
					if res {
						CategoryFilter(ALL, OUT_GET_CURRENT)
						SwitchToMain(inputField, ALL_T)
					} else {
						SwitchToMain(inputField, KEYS)
					}
					MainMutex.Unlock()
				case TORRENT_MOVE:
					MovieTorrent(id, text)
					SwitchToMain(inputField, KEYS)
				case TORRENT_RENAME:
					MainMutex.Lock()
					if tLen > 0 {
						if RenameTorrent(id, t, text) {
							var tDesc string
							for _, tor := range Torrents {
								if tor.Id == id {
									i := strings.Index(tor.Desc, tor.Name)
									tDesc = tor.Desc[:i]
									tor.Name = text
									break
								}
							}
							list.SetItemText(item, tDesc+text, fmt.Sprintf("%d", id))
						}
					}
					SwitchToMain(inputField, KEYS)
					MainMutex.Unlock()
				case TRACKER_ADD:
					if tLen > 0 {
						TrackerAction(id, trackerId,
							text, "trackerAdd")
					}
					updateList()
				case TRACKER_RENAME:
					if tLen > 0 {
						TrackerAction(id, trackerId,
							text, "trackerReplace")
					}
					updateList()
				}
			}
			return event
		})
}

func SetNewCategory(item int, labels string) bool {
	if labels == ALL {
		return false
	}
	var ids []int
	for id, _ := range SelectedIds {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		id := GetId(item, MainList)
		ids = append(ids, id)
	}
	s := strings.Split(labels, ",")
	if labels == DEFAULT {
		s = make([]string, 0)
	}
	// ErrorLog(fmt.Sprintf("labels=%s\n", s))
	for _, i := range ids {
		for _, t := range Torrents {
			if i == t.Id {
				t.Labels = make([]string, 0)
				t.Labels = append(t.Labels, s...)
				break
			}
		}
	}
	var flag, res bool
	for _, c := range s {
		if CurrentCategory == c {
			flag = true
			break
		}
	}
	if !flag {
		max := MainList.GetItemCount()
		var n, i int
		for n, i = range ids {
			for j := 0; j < max; j++ {
				id := GetId(j, MainList)
				if i == id {
					MainList.RemoveItem(j)
					break
				}
			}
		}
		if max == n+1 {
			// CategoryFilter(ALL)
			res = true
		}
	}
	SelectedIds = make(map[int]int)
	type arg struct {
		Labels []string `json:"labels"`
		Ids    []int    `json:"ids"`
	}
	in := &Request{
		Args: arg{
			Labels: s,
			Ids:    ids,
		},
		Method: "torrent-set",
	}
	out := &Response{}
	GetRequest(in, out)
	return res
}

// Debug mode.
func ErrorLog(s string) {
	f, err := os.OpenFile("log.txt",
		os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if _, err = fmt.Fprintln(f, s); err != nil {
		fmt.Fprintln(os.Stderr, err)
		f.Close()
		return
	}
	if err = f.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}

func InitCategory(r int, ctgInfo *tview.List) {
	Category = map[string]int{
		ALL:     Stats.TorrentCount,
		DEFAULT: 0,
	}
	for _, t := range Torrents {
		if len(t.Labels) == 0 {
			Category[DEFAULT]++
		} else {
			for _, c := range t.Labels {
				Category[c]++
			}
		}
	}
	ct := make([]string, 0)
	for k, _ := range Category {
		ct = append(ct, k)
	}
	sort.Strings(ct)
	// All and Default before sorted strings
	ct2 := []string{ALL, DEFAULT}
	for _, k := range ct {
		if k != ALL && k != DEFAULT {
			ct2 = append(ct2, k)
		}
	}
	y := 1 // AddDialogShowCtgDirs() without "All"
	if r == ALL_T {
		y = 0
	}
	for i, k := range ct2 {
		if i >= y {
			ctgInfo.AddItem(fmt.Sprintf("    %s (%d)",
				k, Category[k]), k, 0, nil)
		}
	}
}

func ShowCategoryInfo() {
	MainMutex.Lock()
	MainGrid.RemoveItem(MainList)
	keys := []Key{{"Esc", P("Close")},
		{"F2", P("Set category for selected torrents")},
		{"Enter", P("Filter by category")}}
	SetKeysHeaderText(P("Categories"), FormatKeys(keys), tview.AlignCenter)
	ctgInfo := NewListPrim()
	InitCategory(ALL_T, ctgInfo)
	endwin := func() {
		Status = make(map[string]int)
		ctgInfo.Clear()
		SwitchToMain(ctgInfo, LIST)
		MainMutex.Unlock()
	}
	MainGrid.AddItem(ctgInfo, 2, 0, 1, 3, 0, 0, true)
	App.SetFocus(ctgInfo).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				endwin()
			case tcell.KeyEnter:
				item := ctgInfo.GetCurrentItem()
				_, s := ctgInfo.GetItemText(item)
				CategoryFilter(s, OUT_GET_CURRENT)
				endwin()
			case tcell.KeyF2:
				item := ctgInfo.GetCurrentItem()
				_, s := ctgInfo.GetItemText(item)
				i := MainList.GetCurrentItem()
				SetNewCategory(i, s)
				CategoryFilter(s, OUT_GET_CURRENT)
				endwin()
			}
			return event
		})
}

func CategoryFilter(ctg string, r int) {
	if CurrentCategory == ctg && r != IN_GET_CURRENT {
		return
	}
	CurrentCategory = ctg
	if ctg != ALL {
		CurrentStatus.Name = ALL
		CurrentStatus.Id = STATUS_ALL
	}
	MainList.Clear()
	MainList = NewListPrim()
	for _, t := range Torrents {
		if CheckCategory(&ctg, &t.Labels) {
			MainList.AddItem(t.Desc, fmt.Sprintf("%d", t.Id), 0, nil)
		}
	}
	CategoryStatus.SetText(PrintCtgStat())
}

func CheckCategory(currCtg *string, labels *[]string) bool {
	if *currCtg == ALL || len(*labels) == 0 && *currCtg == DEFAULT {
		return true
	}
	for _, c := range *labels {
		if c == *currCtg {
			return true
		}
	}
	return false
}

func ShowStatusInfo() {
	MainMutex.Lock()
	MainGrid.RemoveItem(MainList)
	keys := []Key{{"Esc", P("Close")}}
	SetKeysHeaderText(P("Status"), FormatKeys(keys), tview.AlignCenter)
	statusInfo := NewListPrim()
	st := []string{ALL, P("Downloading"), P("Queued"), P("Seeding"),
		P("Stopped"), P("Active"), P("Errored")}
	Status = map[string]int{
		st[0]: Stats.TorrentCount,
		st[1]: 0,
		st[2]: 0,
		st[3]: 0,
		st[4]: Stats.PausedTorrentCount,
		st[5]: 0,
		st[6]: 0,
	}
	for _, t := range Torrents {
		if t.DlSpeed > 0 || t.UplSpeed > 0 {
			Status[st[5]]++
		}
		if t.Error != 0 {
			Status[st[6]]++
		}
		switch t.Status {
		case STATUS_SEED:
			Status[st[3]]++
		case STATUS_DOWNLOAD:
			Status[st[1]]++
		case STATUS_DOWNLOAD_WAIT, STATUS_SEED_WAIT:
			Status[st[2]]++
		}
	}
	for _, k := range st {
		statusInfo.AddItem(fmt.Sprintf("    %s (%d)",
			k, Status[k]), fmt.Sprintf("%s", k), 0, nil)
	}
	MainGrid.AddItem(statusInfo, 2, 0, 1, 3, 0, 0, true)
	App.SetFocus(statusInfo).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				Status = make(map[string]int)
				statusInfo.Clear()
				SwitchToMain(statusInfo, LIST)
				MainMutex.Unlock()
			case tcell.KeyEnter:
				StatusFilter(statusInfo, OUT_GET_CURRENT)
				Status = make(map[string]int)
				statusInfo.Clear()
				SwitchToMain(statusInfo, LIST)
				MainMutex.Unlock()
			}
			return event
		})
}

func StatusFilter(statusInfo *tview.List, r int) {
	setItems := func() {
		MainList.Clear()
		MainList = NewListPrim()
		for _, t := range Torrents {
			if CheckStatus(t.Status, t.DlSpeed, t.UplSpeed, t.Error) {
				MainList.AddItem(t.Desc, fmt.Sprintf("%d", t.Id), 0, nil)
			}
		}
		CategoryStatus.SetText(PrintCtgStat())
	}

	if r == IN_GET_CURRENT {
		setItems()
		return
	}
	item := statusInfo.GetCurrentItem()
	_, st := statusInfo.GetItemText(item)
	if CurrentStatus.Name == st && CurrentStatus.Id != STATUS_ACTIVE || Status[st] == 0 {
		return
	}
	CurrentStatus.Name = st
	switch item {
	case 0:
		CurrentStatus.Id = STATUS_ALL
	case 1:
		CurrentStatus.Id = STATUS_DOWNLOAD
	case 2:
		CurrentStatus.Id = STATUS_DOWNLOAD_WAIT
	case 3:
		CurrentStatus.Id = STATUS_SEED
	case 4:
		CurrentStatus.Id = STATUS_STOPPED
	case 5:
		CurrentStatus.Id = STATUS_ACTIVE
	case 6:
		CurrentStatus.Id = STATUS_ERRORED
	}
	if CurrentStatus.Id != STATUS_ALL {
		CurrentCategory = ALL
	}
	setItems()
}

func CheckStatus(status, dl, upl, error int) bool {
	s := CurrentStatus.Id
	var s2 int
	if s == STATUS_DOWNLOAD_WAIT {
		s2 = STATUS_SEED_WAIT
	}
	if s == STATUS_ALL || s == status || s2 != 0 && s2 == status ||
		s == STATUS_ACTIVE && dl > 0 || s == STATUS_ACTIVE && upl > 0 ||
		s == STATUS_ERRORED && error != 0 {
		return true
	}
	return false
}

func TorAction(item int, method string, flag bool) {
	var ids []int
	for id, _ := range SelectedIds {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		id := GetId(item, MainList)
		ids = append(ids, id)
	}
	in := &Request{}
	if method == "torrent-remove" {
		type arg struct {
			DeleteData bool  `json:"delete-local-data"`
			Ids        []int `json:"ids"`
		}
		in = &Request{
			Args: arg{
				DeleteData: flag,
				Ids:        ids,
			},
			Method: method,
		}
	} else {
		type arg struct {
			Ids []int `json:"ids"`
		}
		in = &Request{
			Args: arg{
				Ids: ids,
			},
			Method: method,
		}
	}
	out := &Response{}
	GetRequest(in, out)
}

func SelectAll(list *tview.List, sel bool) {
	MainMutex.Lock()
	max := list.GetItemCount()
	for i := 0; i < max; i++ {
		m, s := list.GetItemText(i)
		id, err := strconv.Atoi(s)
		if err != nil {
			log.Fatal(err)
		}
		if sel {
			SelectedIds[id] = i
			list.SetItemText(i, fmt.Sprintf("[black:yellow]%s", m), s)
		} else {
			if _, ok := SelectedIds[id]; ok {
				delete(SelectedIds, id)
				list.SetItemText(i, fmt.Sprintf("[-:-]%s", m), s)
			}
		}
	}
	MainMutex.Unlock()
}

func SelectItem(list *tview.List) {
	index := list.GetCurrentItem()
	m, s := list.GetItemText(index)
	id, err := strconv.Atoi(s)
	if err != nil {
		log.Fatal(err)
	}
	if _, ok := SelectedIds[id]; ok {
		delete(SelectedIds, id)
		list.SetItemText(index, fmt.Sprintf("[-:-]%s", m), s)
	} else {
		SelectedIds[id] = index
		list.SetItemText(index, fmt.Sprintf("[black:yellow]%s", m), s)
	}
}

func TrackersAdd(id int) *tview.List {
	ti := GetTrackersInfo(id)
	n := len(ti)
	if n == 0 {
		return nil
	}
	max := len(ti[0].Announce)
	for j := 1; j < n; j++ {
		l := len(ti[j].Announce)
		if l > max {
			max = l
		}
	}
	title := fmt.Sprintf("%*s %s", max, P("URL"),
		P("  | Peers | Seeds | Status "))
	Header.SetTextAlign(tview.AlignLeft).SetText(title)
	PrintKeys(TRACKER_ADD)
	trackersInfo := NewListPrim()
	format := func(num int) string {
		if num == 0 || num == -1 {
			return " "
		}
		return fmt.Sprintf("%d", num)
	}
	var i int
	for i = 0; i < n; i++ {
		trackersInfo.AddItem(fmt.Sprintf("%-*s      %5s   %5s   %-30s ",
			max, ti[i].Announce,
			format(ti[i].LastAnnouncePeerCount),
			format(ti[i].SeederCount),
			ti[i].LastAnnounceResult),
			fmt.Sprintf("%d", ti[i].Id), 0, nil)
	}

	return trackersInfo
}

func ShowTrackersInfo(list *tview.List, curItem int) {
	item := MainList.GetCurrentItem()
	id := GetId(item, MainList)
	trackersInfo := TrackersAdd(id)
	if trackersInfo == nil {
		MainMutex.Unlock()
		return
	}
	MainGrid.RemoveItem(list)
	MainGrid.AddItem(trackersInfo, 2, 0, 1, 3, 0, 0, true)
	trackersInfo.SetCurrentItem(curItem)
	App.SetFocus(trackersInfo).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				trackersInfo.Clear()
				SwitchToMain(trackersInfo, LIST)
				MainMutex.Unlock()
			case tcell.KeyF2:
				ShowInputField(trackersInfo, TRACKER_RENAME,
					App.GetInputCapture())
			case tcell.KeyCtrlN:
				ShowInputField(trackersInfo, TRACKER_ADD,
					App.GetInputCapture())
			case tcell.KeyDelete:
				tItem := trackersInfo.GetCurrentItem()
				trackerId := GetId(tItem, trackersInfo)
				TrackerAction(id, trackerId, "", "trackerRemove")
				trackersInfo.RemoveItem(tItem)
			}
			return event
		})
}

func PrintKeys(r int) {
	switch r {
	case CONTENT:
		keys := []Key{{"1/2", P("Priority")}, {P("Space"), P("Get")},
			{"Esc", P("Close")},
			{"F3", P("Next dir")}, {"F4", P("Next root dir")},
			{"F7", P("Search")}, {"Enter", P("Open")}}
		SetKeysHeaderText(fmt.Sprintf("%*s ", StatFmt.Done, P("Done"))+
			P("|   Size    |  Priority  |  Name "),
			FormatKeys(keys), tview.AlignLeft)
	case DIRS:
		keys := []Key{{"Esc", P("Close")}, {"Enter", P("Select dir")},
			{"F2", P("New path")}, {"F7", P("Search")}}
		SetKeysHeaderText(P("Directories"), FormatKeys(keys), tview.AlignCenter)
	case CATEGORY:
		keys := []Key{{"Esc", P("Close")},
			{"Enter", P("Select category")},
			{"F2", P("New category")}, {"F7", P("Search")}}
		SetKeysHeaderText(P("Categories"), FormatKeys(keys), tview.AlignCenter)
	default: // trackers keys
		keys := []Key{{"Esc", P("Close")}, {"F2", P("Edit URL")},
			{"CTRL+N", P("Add a new tracker")},
			{"Delete", P("Remove tracker")}}
		Hotkeys.SetText(FormatKeys(keys))
	}
}

func ShowContentInfo(item int) {
	MainMutex.Lock()
	GetContentInfo(GetId(item, MainList))
	if len(Contents) == 0 {
		return
	}
	MakeContentTree()
	MainGrid.RemoveItem(MainList)
	PrintKeys(CONTENT)
	contentInfo := NewListPrim()
	sec := func(i int) string {
		var sec string
		if ContentsTree[i].RootDir {
			sec = fmt.Sprintf("%d %s", ContentsTree[i].Id, RuneLTeeDir)
		} else if ContentsTree[i].Dir {
			sec = fmt.Sprintf("%d %s", ContentsTree[i].Id, RuneDir)
		} else {
			sec = fmt.Sprintf("%d", ContentsTree[i].Id)
		}
		return sec
	}
	length := len(ContentsTree)
	var lKey, rKey, fKey bool
	var i int
	for i = 0; i < length && ContentsTree[i].Desc != ""; i++ {
		contentInfo.AddItem(fmt.Sprintf("%s %s",
			ContentsTree[i].Desc,
			ContentsTree[i].Name),
			sec(i), 0, nil)
	}
	update := func(max int) {
		for j := 0; j < max; j++ {
			contentInfo.SetItemText(j,
				fmt.Sprintf("%s %s", ContentsTree[j].Desc,
					ContentsTree[j].Name),
				sec(j))
		}
	}
	MainGrid.AddItem(contentInfo, 2, 0, 1, 3, 0, 0, true)
	App.SetFocus(contentInfo).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				Contents = nil
				ContentsTree = nil
				FilePath = ""
				contentInfo.Clear()
				SwitchToMain(contentInfo, LIST)
				MainMutex.Unlock()
			case tcell.KeyEnter:
				it := contentInfo.GetCurrentItem()
				if ContentsTree[it].Progress == 1 {
					OpenItem(FilePath + "/" +
						strings.TrimSuffix(ContentsTree[it].Path, "/"))
				}
			case tcell.KeyF4:
				fKey = true
				fallthrough
			case tcell.KeyF3:
				pos := contentInfo.GetCurrentItem() + 1
				ContentDirSearch(pos, i, contentInfo, fKey)
				fKey = false
			case tcell.KeyF7:
				input := App.GetInputCapture()
				ShowSearchInput(contentInfo, CONTENT, input)
			case tcell.KeyRune:
				switch event.Rune() {
				case ' ':
					ContentWantedAction(item, i, contentInfo)
					update(i)
				case '2':
					rKey = true
					fallthrough
				case '1':
					if rKey == false {
						lKey = true
					}
					ContentPriorityAction(item, i,
						contentInfo, lKey)
					update(i)
					rKey = false
					lKey = false

				}
			}
			return event
		})
}

func ContentDirSearch(i, max int, list *tview.List, key bool) int {
	dir := RuneDir
	if key {
		dir = RuneLTeeDir
	}
	for ; i < max; i++ {
		_, desc := list.GetItemText(i)
		if strings.Contains(desc, dir) {
			list.SetCurrentItem(i)
			i++
			return i
		}
	}
	return 0
}

func SetPrevInput(p tview.Primitive, list *tview.List, r int, input func(event *tcell.EventKey) *tcell.EventKey) {
	MainGrid.RemoveItem(p)
	if r == DIRS || r == CATEGORY {
		MainGrid.AddItem(Hotkeys, 6, 0, 1, 5, 0, 0, false)
	} else {
		MainGrid.AddItem(Hotkeys, 4, 0, 1, 3, 0, 0, false)
	}
	PrintKeys(r)
	App.SetFocus(list).SetInputCapture(input)
}

func ContentPriorityAction(mainItem, count int, contentInfo *tview.List, key bool) {
	var fileIds []int
	var pnum int
	var wg sync.WaitGroup
	defer wg.Wait()
	index := contentInfo.GetCurrentItem()
	_, desc := contentInfo.GetItemText(index)

	PrintContentPriority := func(index, count, pnum int, wg *sync.WaitGroup) {
		for j := index; j < count; j++ {
			if index == 0 || strings.Contains(ContentsTree[j].Path, ContentsTree[index].Path) {
				switch pnum {
				case PRIORITY_NORMAL:
					ContentsTree[j].Priority = PRIORITY_NORMAL
				case PRIORITY_HIGH:
					ContentsTree[j].Priority = PRIORITY_HIGH
				default:
					ContentsTree[j].Priority = PRIORITY_LOW
				}
				PrintContentDesc(j)
			}
		}
		wg.Add(1)
		go PrintContentMixed(index, PRIORITY_SET, count, wg)
	}

	if ContentsTree[index].Priority == PRIORITY_NORMAL {
		if key {
			ContentsTree[index].Priority = PRIORITY_LOW
			pnum = PRIORITY_LOW
		} else {
			ContentsTree[index].Priority = PRIORITY_HIGH
			pnum = PRIORITY_HIGH
		}
		PrintContentDesc(index)
	} else if ContentsTree[index].Priority == PRIORITY_HIGH {
		if key {
			ContentsTree[index].Priority = PRIORITY_NORMAL
			pnum = PRIORITY_NORMAL
			PrintContentDesc(index)
		} else {
			return
		}
	} else {
		if key {
			return
		} else {
			ContentsTree[index].Priority = PRIORITY_NORMAL
			pnum = PRIORITY_NORMAL
			PrintContentDesc(index)
		}
	}
	if !strings.Contains(desc, RuneDir) {
		fileIds = append(fileIds, ContentsTree[index].Id)
		wg.Add(1)
		go PrintContentMixed(index, PRIORITY_SET, count, &wg)
	} else {
		for _, c := range Contents {
			if index == 0 || strings.Contains(c.Name, ContentsTree[index].Path) {
				fileIds = append(fileIds, c.Id)
			}
		}
		PrintContentPriority(index, count, pnum, &wg)
	}
	ContentRpc(mainItem, pnum, PRIORITY_SET, fileIds, true)
}

func ContentRpc(item, pnum, r int, fileIds []int, wanted bool) {
	var q string
	in := &Request{}
	out := &Response{}
	id := GetId(item, MainList)
	s1 := fmt.Sprintf("%d", fileIds)
	s2 := strings.Split(s1, " ")
	strFileIds := strings.Join(s2, ",")
	if r == WANTED_SET {
		if wanted {
			q = "files-wanted"
		} else {
			q = "files-unwanted"
		}
	} else { //PRIORITY_SET
		switch pnum {
		case PRIORITY_HIGH:
			q = "priority-high"
		case PRIORITY_LOW:
			q = "priority-low"
		default:
			q = "priority-normal"
		}
	}
	pdata := []byte(fmt.Sprintf(`{"method":"torrent-set","arguments":{%q:%s,"ids":[%d]}}`, q, strFileIds, id))
	GetRequest(in, out, pdata)
}

func ContentWantedAction(mainItem, count int, contentInfo *tview.List) {
	var fileIds []int
	var wanted bool
	var wg sync.WaitGroup
	defer wg.Wait()
	index := contentInfo.GetCurrentItem()
	_, desc := contentInfo.GetItemText(index)

	PrintContentWanted := func(index, count int, wanted bool, wg *sync.WaitGroup) {
		for j := index; j < count; j++ {
			if index == 0 || strings.Contains(ContentsTree[j].Path, ContentsTree[index].Path) {
				if wanted {
					ContentsTree[j].DlFlag = WANTED
				} else {
					ContentsTree[j].DlFlag = UNWANTED
				}
				PrintContentDesc(j)
			}
		}
		wg.Add(1)
		go PrintContentMixed(index, WANTED_SET, count, wg)
	}

	if ContentsTree[index].DlFlag == WANTED {
		ContentsTree[index].DlFlag = UNWANTED
		PrintContentDesc(index)
	} else {
		ContentsTree[index].DlFlag = WANTED
		PrintContentDesc(index)
		wanted = true
	}
	if !strings.Contains(desc, RuneDir) {
		fileIds = append(fileIds, ContentsTree[index].Id)
		wg.Add(1)
		go PrintContentMixed(index, WANTED_SET, count, &wg)
	} else {
		for _, c := range Contents {
			if index == 0 || strings.Contains(c.Name, ContentsTree[index].Path) {
				fileIds = append(fileIds, c.Id)
			}
		}
		PrintContentWanted(index, count, wanted, &wg)
	}
	ContentRpc(mainItem, 0, WANTED_SET, fileIds, wanted)
}

func ContentMixedFlag(j, index, count int, parentPath *string) bool {
	for i := j + 1; i < count; i++ {
		if strings.Contains(ContentsTree[i].Path, *parentPath) {
			if ContentsTree[i].DlFlag != ContentsTree[index].DlFlag {
				if ContentsTree[i].DlFlag != WANTED_MIXED {
					return true
				}
			}
		} else {
			return false
		}
	}
	return false
}

func ContentMixedSet(i, index, count int, parentPath *string) {
	var flag bool
	if ContentsTree[i].DlFlag != ContentsTree[index].DlFlag {
		if ContentsTree[i].DlFlag == WANTED_MIXED {
			flag = ContentMixedFlag(i, index, count,
				parentPath)
			if flag == false {
				ContentsTree[i].DlFlag = ContentsTree[index].DlFlag
				PrintContentDesc(i)
			}
		} else {
			ContentsTree[i].DlFlag = WANTED_MIXED
			PrintContentDesc(i)
		}
	}
}

func ContentMixedPriorityFlag(j, index, count int, parentPath *string) bool {
	for i := j + 1; i < count; i++ {
		if strings.Contains(ContentsTree[i].Path, *parentPath) &&
			ContentsTree[i].Priority != ContentsTree[index].Priority &&
			ContentsTree[i].Priority != PRIORITY_MIXED {
			return true
		} else {
			return false
		}
	}
	return false
}

func ContentMixedPrioritySet(i, index, count int, parentPath *string) {
	var flag bool
	if ContentsTree[i].Priority != ContentsTree[index].Priority {
		if ContentsTree[i].Priority == PRIORITY_MIXED {
			flag = ContentMixedPriorityFlag(i, index, count,
				parentPath)
			if flag == false {
				ContentsTree[i].Priority = ContentsTree[index].Priority
				PrintContentDesc(i)
			}
		} else {
			ContentsTree[i].Priority = PRIORITY_MIXED
			PrintContentDesc(i)
		}
	}
}

func PrintContentMixed(index, set, count int, wg *sync.WaitGroup) {
	defer wg.Done()
	GetPath := func(j int) string {
		var i int
		end := len(ContentsTree[j].Path) - 2
		for i = end; i >= 0 && ContentsTree[j].Path[i] != '/'; i-- {
			continue
		}
		end = i + 1
		return ContentsTree[j].Path[:end]
	}
	parentPath := GetPath(index)
	for i := index - 1; i >= 0; i-- {
		if ContentsTree[i].Path == parentPath {
			if set == WANTED_SET {
				ContentMixedSet(i, index, count,
					&parentPath)
			} else {
				ContentMixedPrioritySet(i, index, count,
					&parentPath)
			}
			parentPath = GetPath(i)
		}
	}
}

func MakeContentMixed(index int) {
	var wg sync.WaitGroup
	for i := index; i >= 0; i-- {
		if ContentsTree[i].DlFlag != ContentsTree[index].DlFlag {
			wg.Add(1)
			go PrintContentMixed(i, WANTED_SET, index, &wg)
		}
		if ContentsTree[i].Priority != ContentsTree[index].Priority {
			wg.Add(1)
			go PrintContentMixed(i, PRIORITY_SET, index, &wg)
		}
	}
	wg.Wait()
}

func PrintContentDesc(count int) {
	ContentsTree[count].Desc = fmt.Sprintf("  %*s  %10s  %8s    %3s  ",
		StatFmt.Done, FormatProgress(ContentsTree[count].Progress),
		FormatSize(ContentsTree[count].Size),
		FormatPriority(ContentsTree[count].Priority),
		FormatWanted(ContentsTree[count].DlFlag))
}

func MakeContentTree() {
	var count int // Counter of all tokens.
	var end int   // The lastest index.
	max := len(Contents)
	treeMax := max * 2
	ContentsTree = make([]Content, treeMax)
	for i := 0; i < max; i++ {
		end = strings.LastIndex(Contents[i].Name, "/")
		if end != -1 {
			end++
		} else {
			ContentsTree[count] = Contents[i]
			if ContentsTree[count].Size <= 0 {
				ContentsTree[count].Progress = 0
			} else {
				ContentsTree[count].Progress = ContentsTree[count].Progress / float64(ContentsTree[count].Size)
			}
			ContentsTree[count].Path = Contents[i].Name
			PrintContentDesc(count)
			count++
			continue
		}
		count = TreeTokens(i, count, end, treeMax)
	}
	MakeContentMixed(count - 1)
}

func TreeTokens(i, count, end, treeMax int) int {
	var tokBuf string
	tokens := strings.Split(Contents[i].Name, "/")
	spaces := 0
	for j, s := range tokens {
		tokBuf += s + "/"
		pathEq := ComparePath(i, &tokBuf)
		spaces = j + 1
		if count == treeMax {
			m := make([]Content, len(Contents))
			ContentsTree = append(ContentsTree, m...)
		}
		if pathEq == false || i == 0 {
			ContentsTree[count] = Contents[i]
			ContentsTree[count].Name = ""
			AddSpaces(&s, end, spaces, i, count)
			if ContentsTree[count].Dir {
				ContentsTree[count].Name += fmt.Sprintf("[red:]%s[-]",
					tview.Escape(s))
			} else {
				ContentsTree[count].Name += s
			}
			CalcTokSize(i, count, &tokBuf)
			ContentsTree[count].Path = tokBuf

			PrintContentDesc(count)
			count++
		}
	}
	return count
}

// Adds spaces/runes in the filenames of the Contents Tree.
func AddSpaces(s *string, end, spaces, i, k int) {
	var lastName bool
	var lastSymb bool
	max := len(Contents)
	lastFile := max - 1
	if *s == Contents[i].Name[end:] {
		lastName = true
	}
	if lastName && i+1 < max &&
		!strings.Contains(Contents[i+1].Name, Contents[i].Name[:end]) {
		lastSymb = true
	}
	if spaces == 1 {
		ContentsTree[k].Dir = true
	} else if spaces == 2 {
		if i == lastFile {
			ContentsTree[k].Name += " " + RuneLLCorner
		} else {
			if !lastName {
				ContentsTree[k].Dir = true
				ContentsTree[k].RootDir = true
			}
			ContentsTree[k].Name += " " + RuneLTee + " "
		}
	} else {
		if i != lastFile {
			ContentsTree[k].Name += RuneVLine
		} else {
			ContentsTree[k].Name += "   "
		}
		for ; spaces >= 0; spaces-- {
			ContentsTree[k].Name += " "
		}
		if lastName {
			if lastSymb || i == lastFile {
				ContentsTree[k].Name += RuneLLCorner
			} else {
				ContentsTree[k].Name += RuneLTee + " "
			}
		} else {
			ContentsTree[k].Dir = true
		}
	}
}

func CalcTokSize(i, j int, buf *string) {
	var szBytes int64
	var prBytes float64
	count := 0
	max := len(Contents)
	for k := 0; k < max; k++ {
		if strings.Contains(Contents[k].Name, *buf) {
			szBytes += Contents[k].Size
			prBytes += Contents[k].Progress
			count++
		}
	}
	if count == 0 {
		if ContentsTree[j].Size <= 0 {
			ContentsTree[j].Progress = 0
		} else {
			ContentsTree[j].Progress = ContentsTree[j].Progress / float64(ContentsTree[j].Size)
		}
	} else {
		ContentsTree[j].Size = szBytes
		if szBytes <= 0 {
			ContentsTree[j].Progress = 0
		} else {
			ContentsTree[j].Progress = prBytes / float64(szBytes)
		}
	}
}

func ComparePath(i int, buf *string) bool {
	for k := 0; k < i; k++ {
		if strings.Contains(Contents[k].Name, *buf) {
			return true
		}
	}
	return false
}

func PrintPeers(quit <-chan bool, id int) {
	for {
		select {
		case <-quit:
			return
		default:
			PeersMutex.Lock()
			GetSessionStats()
			Peers.Clear()
			pi := GetPeersInfo(id)
			for _, p := range pi {
				fmt.Fprintf(Peers,
					" %20s  %6s    %10s   %10s  %10s"+
						"     %-35s \n",
					p.Address,
					FormatProgress(p.Progress),
					FormatSpeed(p.DownloadSpeed),
					FormatSpeed(p.UploadSpeed),
					p.FlagStr, p.ClientName)
			}
			ShowStatusbar()
			App.Draw()
			PeersMutex.Unlock()
			time.Sleep(UpdateInt * time.Second)
		}
	}
}

func ShowPeersInfo(item int) {
	MainMutex.Lock()
	MainGrid.RemoveItem(MainList)
	title := fmt.Sprintf("%*s %s", 18, "IP",
		P(" |  Done  | Downloading | Uploading |   Flags   | Client"))
	keys := []Key{{"Esc", P("Close")}, {"F2", P("(Un)pause updates")}}
	SetKeysHeaderText(title, FormatKeys(keys), tview.AlignLeft)
	Peers = NewTextPrim(" ").SetWrap(false)
	MainGrid.AddItem(Peers, 2, 0, 1, 3, 0, 0, true)
	var pause bool
	quit := make(chan bool)
	id := GetId(item, MainList)
	go PrintPeers(quit, id)
	App.SetFocus(Peers).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				quit <- true
				Peers.Clear()
				SwitchToMain(Peers, LIST)
				MainMutex.Unlock()
			case tcell.KeyF2:
				if !pause {
					PeersMutex.Lock()
					pause = true
				} else {
					PeersMutex.Unlock()
					pause = false
				}
			}
			return event
		})
}

func SearchItem(s string, i int, list *tview.List) (int, int) {
	var res string
	s = strings.ToLower(s)
	max := list.GetItemCount()
	for ; i < max; i++ {
		res, _ = list.GetItemText(i)
		res = strings.ToLower(res)
		if strings.Contains(res, s) {
			list.SetCurrentItem(i)
			i++
			return i, max
		}
	}
	return 0, max
}

func ShowSearchInput(list *tview.List, fromList int, input func(event *tcell.EventKey) *tcell.EventKey) {
	var pos, max int
	MainGrid.RemoveItem(Hotkeys)
	keys := []Key{{"Esc", P("Cancel")}, {"F3", P("Next")}}
	inputField := NewInputFieldPrim(FormatKeys(keys) + P("Search:"))
	if fromList == DIRS || fromList == CATEGORY {
		MainGrid.AddItem(inputField, 6, 0, 1, 4, 0, 0, false)
	} else {
		MainGrid.AddItem(inputField, 4, 0, 1, 3, 0, 0, false)
	}
	endwin := func() {
		switch fromList {
		case CONTENT, DIRS, CATEGORY:
			SetPrevInput(inputField, list, fromList, input)
		default:
			SwitchToMain(inputField, KEYS)
		}
	}
	App.SetFocus(inputField).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				endwin()
			case tcell.KeyEnter:
				text := inputField.GetText()
				pos, _ = SearchItem(text, pos, list)
				endwin()
			case tcell.KeyF3:
				text := inputField.GetText()
				pos, max = SearchItem(text, pos, list)
				if pos == max {
					pos = 0
				}
			}
			return event
		})
}

func SwitchToMain(p tview.Primitive, item int) {
	MainGrid.RemoveItem(p)
	if item == KEYS {
		MainGrid.AddItem(Hotkeys, 4, 0, 1, 3, 0, 0, false)
	} else if item == LIST {
		MainGrid.AddItem(MainList, 2, 0, 1, 3, 0, 0, true)
	} else {
		MainGrid.AddItem(Hotkeys, 4, 0, 1, 3, 0, 0, false)
		MainGrid.AddItem(MainList, 2, 0, 1, 3, 0, 0, true)
	}
	SetMainInput()
}

func PrintCtgStat() string {
	return fmt.Sprintf(P("Status")+": %s                         "+
		P("Category")+": %s", CurrentStatus.Name, CurrentCategory)
}

func GetId(item int, list *tview.List) int {
	_, secondary := list.GetItemText(item)
	id, err := strconv.Atoi(secondary)
	if err != nil {
		log.Fatal(err)
	}
	return id
}

func ShowGeneralInfo(item int) {
	MainMutex.Lock()
	gi := GetGeneralInfo(GetId(item, MainList))
	MainGrid.RemoveItem(MainList)
	keys := []Key{{"Esc", P("Close")}}
	SetKeysHeaderText(P("General Info"), FormatKeys(keys), tview.AlignCenter)
	pre := [...]string{P("Name"), "ID", P("Hash"), P("Category"),
		P("Location"), P("Comment"), P("Uploaded"), P("Ratio"),
		P("Created"), P("Creator"), P("Added"), P("Total Size"),
		P("Errors")}
	n := 0
	for _, s := range pre {
		l := utf8.RuneCountInString(s)
		if l > n {
			n = l
		}
	}
	text := fmt.Sprintf("%*s: %s\n"+"%*s: %d\n"+"%*s: %s\n"+
		"%*s: %s\n"+"%*s: %s\n"+"%*s: %s\n\n"+"%*s: %s\n"+
		"%*s: %g\n"+"%*s: %s\n"+"%*s: %s\n"+"%*s: %s\n"+
		"%*s: %s\n"+"%*s: %s\n",
		n, pre[0], gi[0].Name, n, pre[1], gi[0].Id,
		n, pre[2], gi[0].HashString,
		n, pre[3], strings.Join(gi[0].Labels, ","),
		n, pre[4], gi[0].DownloadDir, n, pre[5], gi[0].Comment,
		n, pre[6], FormatSize(gi[0].UploadedEver),
		n, pre[7], FormatRatio(gi[0].UploadRatio),
		n, pre[8], FormatDate(gi[0].DateCreated),
		n, pre[9], gi[0].Creator,
		n, pre[10], FormatDate(gi[0].AddedDate),
		n, pre[11], FormatSize(gi[0].TotalSize),
		n, pre[12], gi[0].ErrorString)
	genInfo := NewTextPrim(text)
	MainGrid.AddItem(genInfo, 2, 0, 1, 3, 0, 0, true)
	App.SetFocus(genInfo).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				genInfo.Clear()
				SwitchToMain(genInfo, LIST)
				MainMutex.Unlock()
			}
			return event
		})
}

func ShowCurrent() {
	var prev int
	for {
		MainMutex.Lock()
		prev = Stats.TorrentCount
		GetSessionStats()
		GetTorrentsInfo()
		if Stats.TorrentCount != prev {
			GetTorrents()
			UpdateNewTorrents()
		}
		UpdateCurrentTorrents()
		ShowStatusbar()
		App.Draw()
		MainMutex.Unlock()
		time.Sleep(UpdateInt * time.Second)
	}
}

func UpdateNewTorrents() {
	item := MainList.GetCurrentItem()
	MainGrid.RemoveItem(App.GetFocus())
	MainGrid.RemoveItem(MainList)
	MainList.Clear()
	InitMainList()
	n := MainList.GetItemCount()
	if item <= n && n != 0 {
		MainList.SetCurrentItem(item)
	}
	SwitchToMain(Hotkeys, ALL_T)
}

func UpdateCurrentTorrents() {
	var cfmt string
	n := MainList.GetItemCount()
	i := 0
	for _, t := range Torrents {
		if CheckStatCtg(t.Status, t.DlSpeed, t.UplSpeed, t.Error, &t.Labels) {
			if _, ok := SelectedIds[t.Id]; ok {
				cfmt = "[black:yellow]%s"
			} else {
				cfmt = "%s"
			}
			if i < n {
				MainList.SetItemText(i,
					fmt.Sprintf(cfmt, t.Desc), fmt.Sprintf("%d", t.Id))
				i++
			}
		}
	}

	if n != i {
		item := MainList.GetCurrentItem()
		if CurrentStatus.Id == STATUS_ALL && CurrentCategory == ALL || i == 0 {
			if i == 0 {
				CurrentStatus.Id = STATUS_ALL
				CurrentStatus.Name = ALL
				CurrentCategory = ALL
				CategoryStatus.SetText(PrintCtgStat())
			}
			UpdateNewTorrents()
			return
		} else if CurrentStatus.Id == STATUS_ALL && CurrentCategory != ALL {
			CategoryFilter(CurrentCategory, IN_GET_CURRENT)

		} else {
			StatusFilter(nil, IN_GET_CURRENT)
		}
		if item <= i && i != 0 {
			MainList.SetCurrentItem(item)
		}
		MainGrid.AddItem(MainList, 2, 0, 1, 3, 0, 0, true)
		SetMainInput()
	}
}

func CheckStatCtg(status, dl, upl, error int, labels *[]string) bool {
	var res bool
	if CurrentStatus.Id == STATUS_ALL && CurrentCategory == ALL {
		return true
	} else if CurrentStatus.Id == STATUS_ALL && CurrentCategory != ALL {
		res = CheckCategory(&CurrentCategory, labels)
	} else {
		res = CheckStatus(status, dl, upl, error)
	}
	return res
}

func ShowStatusbar() {
	Statusbar.SetText(fmt.Sprintf(ALL+": %d | "+P("Resumed")+": %d"+
		" | "+P("Paused")+": %d | "+P("Downloading")+": %s | "+
		P("Uploading")+": %s ",
		Stats.TorrentCount, Stats.ActiveTorrentCount,
		Stats.PausedTorrentCount, FormatSpeed(Stats.DownloadSpeed),
		FormatSpeed(Stats.UploadSpeed)))
}

func InitMainList() {
	MainList = NewListPrim()
	for _, t := range Torrents {
		MainList.AddItem(t.Desc, fmt.Sprintf("%d", t.Id), 0, nil)
	}
}

func SetKeysHeaderText(title, keys string, align int) {
	Header.SetTextAlign(align).SetText(title)
	Hotkeys.SetText(keys)
}

func FormatKeys(keys []Key) string {
	res := "[-:-]"
	for _, k := range keys {
		res += k.Name + "[#E6DB58:#3465A4]" + " " + k.Desc + " [-:-]"
	}
	return res
}

func FormatWanted(wanted int) string {
	switch wanted {
	case WANTED:
		return "[*]"
	case UNWANTED:
		return "[ ]"
	default:
		return "[-[]" // Without applying color effect,
		// need to put an opening square bracket before the closing square bracket.
	}
}

func FormatWantedPre(wanted bool) int {
	if wanted {
		return WANTED
	} else {
		return UNWANTED
	}
}

func FormatPriority(priority int) string {
	switch priority {
	case PRIORITY_LOW:
		return "Low"
	case PRIORITY_HIGH:
		return "High"
	case PRIORITY_MIXED:
		return "Mixed"
	default:
		return "Normal"
	}
}

func FormatRatio(r float64) float64 {
	if r == -1 {
		return 0
	}
	return r
}

func FormatDate(utime int64) string {
	t := time.Unix(utime, 0)
	return fmt.Sprintf("%s", t.Format("2006-01-02 15:04:05"))
}

func FormatPeers(p int) string {
	if p == 0 {
		return " "
	}
	return fmt.Sprintf("%d", p)
}

func FormatProgress(pr float64) string {
	pr *= 100
	n := 2
	if pr >= 10 {
		n = 3
	} else if pr < 1 {
		n = 1
	}
	return fmt.Sprintf("%1.*g%%", n, pr)
}

func FormatStatus(status, err int) string {
	if err != 0 {
		return StatSymb.Errored
	}
	switch status {
	case STATUS_STOPPED:
		return StatSymb.Stopped
	case STATUS_CHECK_WAIT:
		return StatSymb.CheckWait
	case STATUS_CHECK:
		return StatSymb.Check
	case STATUS_DOWNLOAD_WAIT, STATUS_SEED_WAIT:
		return StatSymb.DlWait
	case STATUS_DOWNLOAD:
		return StatSymb.Dl
	case STATUS_SEED:
		return StatSymb.Seed
	default:
		return "? "
	}
}

func FormatSize(bytes int64) string {
	size := float64(bytes) / MB
	nfmt := func(size *float64) int {
		n := 2
		if *size >= 998 {
			n = 4
		} else if *size >= 98 {
			n = 3
		}
		return n
	}
	switch {
	case size > GB:
		return fmt.Sprintf("%3.3g "+P("GiB"), size/GB)
	case size >= 1:
		return fmt.Sprintf("%2.*g "+P("MiB"), nfmt(&size), size)
	case bytes >= 1024:
		size = float64(bytes) / 1024
		return fmt.Sprintf("%2.*g "+P("KiB"), nfmt(&size), size)
	default:
		return fmt.Sprintf("%d "+P("B"), bytes)
	}
}

func FormatSpeed(bps int) string {
	if bps == 0 {
		return " "
	}
	speed := float64(bps) / SPEED_KB
	if speed > SPEED_KB {
		return fmt.Sprintf("%1.1f "+P("MB/s"), speed/SPEED_KB)
	} else {
		return fmt.Sprintf("%1.1f "+P("kB/s"), speed)
	}
}

func FormatEta(seconds int64) string {
	if seconds < 0 {
		return " "
	}
	eta := seconds / 3600
	switch {
	case eta >= 24:
		return fmt.Sprintf("%d"+P("d"), eta/60)
	case eta >= 1:
		return fmt.Sprintf("%d"+P("h"), eta)
	case seconds >= 60:
		return fmt.Sprintf("%d"+P("m"), seconds/60)
	default:
		return fmt.Sprintf("%d"+P("s"), seconds)
	}
}

func GetVersion() int {
	type SessionSettings struct {
		Version string `json:"version,omitempty"`
	}
	in := &Request{
		Method: "session-get",
	}
	out := &Response{Args: &SessionSettings{}}
	GetRequest(in, out)
	ver := out.Args.(*SessionSettings).Version
	v, err := strconv.Atoi(ver[:1])
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func GetCtgDirs() {
	in := &Request{
		Args: Arg{
			Fields: []string{"id", "labels", "downloadDir"},
		},
		Method: "torrent-get",
	}
	out := &Response{Args: &TorrentsGet{}}
	GetRequest(in, out)
	Torrents = out.Args.(*TorrentsGet).All
}

func GetAction(id int, r string) string {
	in := &Request{
		Args: Arg{
			Fields: []string{r},
			Ids:    []int{id},
		},
		Method: "torrent-get",
	}
	if r == "comment" {
		type Comment struct {
			Url string `json:"comment"`
		}
		type Get struct {
			All []Comment `json:"torrents"`
		}
		out := &Response{Args: &Get{}}
		GetRequest(in, out)
		return out.Args.(*Get).All[0].Url
	} else {
		type Dir struct {
			Dir string `json:"downloadDir"`
		}
		type Get struct {
			All []Dir `json:"torrents"`
		}
		out := &Response{Args: &Get{}}
		GetRequest(in, out)
		return out.Args.(*Get).All[0].Dir
	}
}

func GetTrackersInfo(id int) []TrackersInfo {
	type TorrentTrackers struct {
		Trackers []TrackersInfo `json:"trackerStats"`
	}

	type TorrentsGetTrackersInfo struct {
		Torrents []TorrentTrackers `json:"torrents"`
	}
	in := &Request{
		Args: Arg{
			Fields: []string{"trackerStats"},
			Ids:    []int{id},
		},
		Method: "torrent-get",
	}
	out := &Response{Args: &TorrentsGetTrackersInfo{}}
	GetRequest(in, out)
	return out.Args.(*TorrentsGetTrackersInfo).Torrents[0].Trackers
}

func GetContentInfo(id int) {
	type TorrentsGetContentInfo struct {
		Torrents []TorrentContent `json:"torrents"`
	}
	in := &Request{
		Args: Arg{
			Fields: []string{"downloadDir",
				"fileStats", "files"},
			Ids: []int{id},
		},
		Method: "torrent-get",
	}
	out := &Response{Args: &TorrentsGetContentInfo{}}
	GetRequest(in, out)
	files := out.Args.(*TorrentsGetContentInfo).Torrents[0].Files
	fileStats := out.Args.(*TorrentsGetContentInfo).Torrents[0].FileStats
	FilePath = out.Args.(*TorrentsGetContentInfo).Torrents[0].Path

	length := len(files)
	Contents = make([]Content, length)
	for i := 0; i < length; i++ {
		Contents[i].Name = files[i].Name
		Contents[i].Progress = files[i].Progress
		Contents[i].Size = files[i].Size
		Contents[i].Priority = fileStats[i].Priority
		Contents[i].DlFlag = FormatWantedPre(fileStats[i].DlFlag)
		Contents[i].Id = i
	}
	sort.Slice(Contents, func(i, j int) bool {
		return Contents[i].Name < Contents[j].Name
	})
}

func GetPeersInfo(id int) []PeersInfo {
	type TorrentPeers struct {
		Peers []PeersInfo `json:"peers"`
	}
	type TorrentsGetPeersInfo struct {
		Torrents []TorrentPeers `json:"torrents"`
	}
	in := &Request{
		Args: Arg{
			Fields: []string{"peers"},
			Ids:    []int{id},
		},
		Method: "torrent-get",
	}
	out := &Response{Args: &TorrentsGetPeersInfo{}}
	GetRequest(in, out)
	return out.Args.(*TorrentsGetPeersInfo).Torrents[0].Peers
}

func GetGeneralInfo(id int) []*GeneralInfo {
	type TorrentsGetGeneralInfo struct {
		All []*GeneralInfo `json:"torrents"`
	}
	in := &Request{
		Args: Arg{
			Fields: []string{"name", "id", "uploadRatio",
				"uploadedEver", "hashString",
				"downloadDir", "comment", "creator",
				"dateCreated", "addedDate", "totalSize",
				"errorString", "labels"},
			Ids: []int{id},
		},
		Method: "torrent-get",
	}
	out := &Response{Args: &TorrentsGetGeneralInfo{}}
	GetRequest(in, out)
	return out.Args.(*TorrentsGetGeneralInfo).All
}

func GetSessionStats() {
	in := &Request{
		Method: "session-stats",
	}
	out := &Response{Args: &SessionStats{}}
	GetRequest(in, out)
	Stats = out.Args.(*SessionStats)
}

func GetTorrents() {
	in := &Request{
		Args: Arg{
			Fields: []string{"id", "name", "labels", "addedDate"},
		},
		Method: "torrent-get",
	}
	out := &Response{Args: &TorrentsGet{}}
	GetRequest(in, out)
	Torrents = out.Args.(*TorrentsGet).All
	sort.Slice(Torrents, func(i, j int) bool {
		return strings.ToLower(Torrents[i].Name) < strings.ToLower(Torrents[j].Name)
	})
	GetTorrentsInfo()
}

func GetTorrentsInfo() {
	in := &Request{
		Args: Arg{
			Fields: []string{"id", "sizeWhenDone", "error",
				"percentDone", "status", "peersConnected",
				"rateDownload", "rateUpload", "eta"},
		},
		Method: "torrent-get",
	}
	out := &Response{Args: &TorrentsGetInfo{}}
	GetRequest(in, out)
	TorrentsInfo := out.Args.(*TorrentsGetInfo).All
	for _, t := range TorrentsInfo {
		for _, s := range Torrents {
			if s.Id == t.Id {
				s.Desc = fmt.Sprintf(" %*s    %11s   %11s"+
					"   %5s   %6s  %10s  %s %s",
					StatFmt.Eta, FormatEta(t.Eta),
					FormatSpeed(t.UplSpeed),
					FormatSpeed(t.DlSpeed),
					FormatPeers(t.Peers),
					FormatProgress(t.Progress),
					FormatSize(t.Size),
					FormatStatus(t.Status, t.Error),
					s.Name)
				s.Status = t.Status
				s.Progress = t.Progress
				s.Size = t.Size
				s.DlSpeed = t.DlSpeed
				s.UplSpeed = t.UplSpeed
				s.Error = t.Error
			}
		}
	}
}

func GetRequest(in *Request, out *Response, s ...[]byte) {
	var pdata []byte
	if len(s) == 0 {
		var err error
		pdata, err = json.Marshal(in)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		pdata = s[0]
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", URL, bytes.NewReader(pdata))
	if err != nil {
		log.Fatal(err)
	}
	header := "X-Transmission-Session-Id"
	req.Header.Set(header, HeaderId)
	if AuthUsername != "" || AuthPasswd != "" {
		req.SetBasicAuth(AuthUsername, AuthPasswd)
	}
	resp, err := client.Do(req)
	if err != nil {
		if App != nil {
			App.Stop()
		}
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer func() {
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	if resp.StatusCode == 409 {
		HeaderId = resp.Header.Get(header)
		if len(s) == 0 {
			GetRequest(in, out)
		} else {
			GetRequest(in, out, s[0])
		}
		return
	} else if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "%d: %s\n", resp.StatusCode,
			http.StatusText(resp.StatusCode))
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, out)
	if err != nil {
		log.Fatal(err)
	}
}

var lng *message.Printer

func P(text string) string {
	return lng.Sprintf(text)
}

func SetLocales() {
	l := os.Getenv("LANG")
	if l == "" || l == "C" {
		os.Setenv("LANG", "en")
	}
	if os.Getenv("LC_NUMERIC") == "" {
		os.Setenv("LC_NUMERIC", "C")
	}
	SetLocale.SetLocale(SetLocale.LC_ALL, "")
	SetLocale.SetLocale(SetLocale.LC_NUMERIC, "C")

	matcher := language.NewMatcher(message.DefaultCatalog.Languages())
	lang := strings.Split(os.Getenv("LANG"), ".")
	langTag, _, _ := matcher.Match(language.MustParse(lang[0]))
	lng = message.NewPrinter(langTag)
}
