// Code generated by running "go generate" in golang.org/x/text. DO NOT EDIT.

package main

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type dictionary struct {
	index []uint32
	data  string
}

func (d *dictionary) Lookup(key string) (data string, ok bool) {
	p := messageKeyToIndex[key]
	start, end := d.index[p], d.index[p+1]
	if start == end {
		return "", false
	}
	return d.data[start:end], true
}

func init() {
	dict := map[string]catalog.Dictionary{
		"en": &dictionary{index: enIndex, data: enData},
		"ru": &dictionary{index: ruIndex, data: ruData},
	}
	fallback := language.MustParse("en")
	cat, err := catalog.NewFromMap(dict, catalog.Fallback(fallback))
	if err != nil {
		panic(err)
	}
	message.DefaultCatalog = cat
}

var messageKeyToIndex = map[string]int{
	"   Added Date":               42,
	"   Name":                     43,
	"   Name ":                    27,
	"   Progress":                 44,
	"   Size":                     45,
	"  Done  |  Size   |  Name ":  79,
	"  | Peers | Seeds | Status ": 93,
	" Category: ":                 51,
	" Path":                       50,
	" Size":                       56,
	" Start torrent:":             53,
	" |  Done  | Downloading | Uploading |   Flags   | Client": 106,
	"(Un)expand dir":    59,
	"(Un)pause updates": 107,
	"<0,1,2,3,...> Mark files for download by index numbers":      5,
	"<filename-or-URL>  Add torrent":                              4,
	"<name1,name2,...>  Set categories when adding a new torrent": 3,
	"<path>  Set download dir when adding a new torrent":          2,
	"Active":                        91,
	"Add a new tracker":             104,
	"Add torrent":                   52,
	"Added":                         119,
	"All":                           14,
	"B":                             128,
	"Cancel":                        49,
	"Categories":                    90,
	"Category":                      29,
	"Check wait":                    17,
	"Checking":                      18,
	"Close":                         39,
	"Comment":                       114,
	"Content":                       34,
	"Created":                       117,
	"Creator":                       118,
	"Default":                       15,
	"Directories":                   100,
	"Do you really want to delete":  83,
	"Done":                          24,
	"Downloading":                   20,
	"ETA":                           25,
	"Edit URL":                      103,
	"Enter a new category name(s):": 47,
	"Enter a new path:":             48,
	"Enter announce URL:":           86,
	"Errored":                       22,
	"Errors":                        121,
	"Filter by category":            89,
	"Free":                          46,
	"General":                       30,
	"General Info":                  110,
	"Get":                           58,
	"GiB":                           125,
	"Hash":                          112,
	"Help":                          36,
	"Hotkeys":                       63,
	"KiB":                           127,
	"Location":                      113,
	"MB/s":                          129,
	"MiB":                           126,
	"Move":                          35,
	"Move to:":                      84,
	"Name":                          111,
	"New category":                  102,
	"New path":                      99,
	"Next":                          108,
	"Next dir":                      95,
	"Next root dir":                 96,
	"No":                            81,
	"Open":                          80,
	"Path":                          61,
	"Paused":                        123,
	"Peers":                         32,
	"Print current version":         13,
	"Print tracker URLs of a torrent file to standard output": 11,
	"Priority":                           94,
	"Queued":                             19,
	"Quit":                               28,
	"Ratio":                              116,
	"Remove tracker":                     105,
	"Rename to:":                         85,
	"Resumed":                            122,
	"Search":                             33,
	"Search:":                            109,
	"Seeding":                            21,
	"Select category":                    101,
	"Select dir":                         98,
	"Set category for selected torrents": 88,
	"Set host":                           0,
	"Set password":                       7,
	"Set port":                           1,
	"Set the interval for updating torrents information in seconds": 12,
	"Set username": 6,
	"Show dialog when adding a new torrent file (not url/magnet)": 10,
	"Show full status names": 8,
	"Sort":                   40,
	"Sort by":                41,
	"SortBy":                 37,
	"Space":                  57,
	"Start added torrent":    9,
	"Start yes/no":           60,
	"Status":                 23,
	"Stopped":                16,
	"Torrent already added":  62,
	"Total Size":             120,
	"Tracker URL:":           87,
	"Trackers":               31,
	"URL":                    92,
	"Uploaded":               115,
	"Uploading":              124,
	"Yes":                    82,
	"You need transmission-daemon version 3.00 or later for the categories support.": 38,
	"cancel selection": 74,
	"create a new category for selected torrent(s)": 75,
	"d":                                 131,
	"h":                                 132,
	"kB/s":                              130,
	"m":                                 133,
	"no":                                54,
	"open comment url":                  76,
	"open download dir":                 77,
	"or":                                69,
	"preview/open file(s)":              71,
	"reannounce":                        67,
	"remove torrent(s)":                 68,
	"remove torrent(s) with data":       70,
	"rename torrent":                    78,
	"s":                                 134,
	"select all":                        73,
	"select/unselect":                   72,
	"start":                             64,
	"stop":                              65,
	"verify":                            66,
	"yes":                               55,
	"|   Size    |  Priority  |  Name ": 97,
	"|  Uploading  | Downloading | Peers |  Done  |   Size    |": 26,
}

var enIndex = []uint32{ // 136 elements
	// Entry 0 - 1F
	0x00000000, 0x00000009, 0x00000012, 0x00000045,
	0x00000081, 0x000000a0, 0x000000d7, 0x000000e4,
	0x000000f1, 0x00000108, 0x0000011c, 0x00000158,
	0x00000190, 0x000001ce, 0x000001e4, 0x000001e8,
	0x000001f0, 0x000001f8, 0x00000203, 0x0000020c,
	0x00000213, 0x0000021f, 0x00000227, 0x0000022f,
	0x00000236, 0x0000023b, 0x0000023f, 0x0000027a,
	0x00000287, 0x0000028c, 0x00000295, 0x0000029d,
	// Entry 20 - 3F
	0x000002a6, 0x000002ac, 0x000002b3, 0x000002bb,
	0x000002c0, 0x000002c5, 0x000002cc, 0x0000031b,
	0x00000321, 0x00000326, 0x0000032e, 0x00000340,
	0x0000034c, 0x0000035c, 0x00000368, 0x0000036d,
	0x0000038b, 0x0000039d, 0x000003a4, 0x000003ae,
	0x000003be, 0x000003ca, 0x000003de, 0x000003e1,
	0x000003e5, 0x000003ef, 0x000003f5, 0x000003f9,
	0x00000408, 0x00000415, 0x0000041a, 0x00000430,
	// Entry 40 - 5F
	0x00000438, 0x0000043e, 0x00000443, 0x0000044a,
	0x00000455, 0x00000467, 0x0000046a, 0x00000486,
	0x0000049b, 0x000004ab, 0x000004b6, 0x000004c7,
	0x000004f5, 0x00000506, 0x00000518, 0x00000527,
	0x00000546, 0x0000054b, 0x0000054e, 0x00000552,
	0x0000056f, 0x00000578, 0x00000583, 0x00000597,
	0x000005a4, 0x000005c7, 0x000005da, 0x000005e5,
	0x000005ec, 0x000005f0, 0x00000610, 0x00000619,
	// Entry 60 - 7F
	0x00000622, 0x00000630, 0x00000656, 0x00000661,
	0x0000066a, 0x00000676, 0x00000686, 0x00000693,
	0x0000069c, 0x000006ae, 0x000006bd, 0x000006fa,
	0x0000070c, 0x00000711, 0x00000719, 0x00000726,
	0x0000072b, 0x00000730, 0x00000739, 0x00000741,
	0x0000074a, 0x00000750, 0x00000758, 0x00000760,
	0x00000766, 0x00000771, 0x00000778, 0x00000780,
	0x00000787, 0x00000791, 0x00000795, 0x00000799,
	// Entry 80 - 9F
	0x0000079d, 0x0000079f, 0x000007a4, 0x000007a9,
	0x000007ab, 0x000007ad, 0x000007af, 0x000007b1,
} // Size: 568 bytes

const enData string = "" + // Size: 1969 bytes
	"\x02Set host\x02Set port\x02<path>  Set download dir when adding a new t" +
	"orrent\x02<name1,name2,...>  Set categories when adding a new torrent" +
	"\x02<filename-or-URL>  Add torrent\x02<0,1,2,3,...> Mark files for downl" +
	"oad by index numbers\x02Set username\x02Set password\x02Show full status" +
	" names\x02Start added torrent\x02Show dialog when adding a new torrent f" +
	"ile (not url/magnet)\x02Print tracker URLs of a torrent file to standard" +
	" output\x02Set the interval for updating torrents information in seconds" +
	"\x02Print current version\x02All\x02Default\x02Stopped\x02Check wait\x02" +
	"Checking\x02Queued\x02Downloading\x02Seeding\x02Errored\x02Status\x02Don" +
	"e\x02ETA\x02|  Uploading  | Downloading | Peers |  Done  |   Size    |" +
	"\x04\x03   \x01 \x05\x02Name\x02Quit\x02Category\x02General\x02Trackers" +
	"\x02Peers\x02Search\x02Content\x02Move\x02Help\x02SortBy\x02You need tra" +
	"nsmission-daemon version 3.00 or later for the categories support.\x02Cl" +
	"ose\x02Sort\x02Sort by\x04\x03   \x00\x0b\x02Added Date\x04\x03   \x00" +
	"\x05\x02Name\x04\x03   \x00\x09\x02Progress\x04\x03   \x00\x05\x02Size" +
	"\x02Free\x02Enter a new category name(s):\x02Enter a new path:\x02Cancel" +
	"\x04\x01 \x00\x05\x02Path\x04\x01 \x01 \x0a\x02Category:\x02Add torrent" +
	"\x04\x01 \x00\x0f\x02Start torrent:\x02no\x02yes\x04\x01 \x00\x05\x02Siz" +
	"e\x02Space\x02Get\x02(Un)expand dir\x02Start yes/no\x02Path\x02Torrent a" +
	"lready added\x02Hotkeys\x02start\x02stop\x02verify\x02reannounce\x02remo" +
	"ve torrent(s)\x02or\x02remove torrent(s) with data\x02preview/open file(" +
	"s)\x02select/unselect\x02select all\x02cancel selection\x02create a new " +
	"category for selected torrent(s)\x02open comment url\x02open download di" +
	"r\x02rename torrent\x04\x02  \x01 \x18\x02Done  |  Size   |  Name\x02Ope" +
	"n\x02No\x02Yes\x02Do you really want to delete\x02Move to:\x02Rename to:" +
	"\x02Enter announce URL:\x02Tracker URL:\x02Set category for selected tor" +
	"rents\x02Filter by category\x02Categories\x02Active\x02URL\x04\x02  \x01" +
	" \x19\x02| Peers | Seeds | Status\x02Priority\x02Next dir\x02Next root d" +
	"ir\x04\x00\x01 !\x02|   Size    |  Priority  |  Name\x02Select dir\x02Ne" +
	"w path\x02Directories\x02Select category\x02New category\x02Edit URL\x02" +
	"Add a new tracker\x02Remove tracker\x04\x01 \x008\x02|  Done  | Download" +
	"ing | Uploading |   Flags   | Client\x02(Un)pause updates\x02Next\x02Sea" +
	"rch:\x02General Info\x02Name\x02Hash\x02Location\x02Comment\x02Uploaded" +
	"\x02Ratio\x02Created\x02Creator\x02Added\x02Total Size\x02Errors\x02Resu" +
	"med\x02Paused\x02Uploading\x02GiB\x02MiB\x02KiB\x02B\x02MB/s\x02kB/s\x02" +
	"d\x02h\x02m\x02s"

var ruIndex = []uint32{ // 136 elements
	// Entry 0 - 1F
	0x00000000, 0x0000001e, 0x0000003c, 0x000000aa,
	0x00000116, 0x00000156, 0x000001bd, 0x000001f2,
	0x00000214, 0x00000252, 0x0000028d, 0x00000304,
	0x00000370, 0x000003ea, 0x00000408, 0x0000040f,
	0x00000427, 0x0000043c, 0x00000456, 0x0000046d,
	0x0000047f, 0x00000490, 0x000004a3, 0x000004b5,
	0x000004c2, 0x000004cf, 0x000004da, 0x00000533,
	0x00000542, 0x0000054d, 0x00000560, 0x0000056b,
	// Entry 20 - 3F
	0x0000057a, 0x00000583, 0x0000058e, 0x00000599,
	0x000005b0, 0x000005bd, 0x000005d2, 0x0000064d,
	0x0000065c, 0x00000671, 0x0000068d, 0x000006b2,
	0x000006c0, 0x000006d8, 0x000006ec, 0x000006fd,
	0x00000735, 0x00000758, 0x00000765, 0x00000773,
	0x0000078d, 0x000007ad, 0x000007d7, 0x000007de,
	0x000007e3, 0x000007f5, 0x00000802, 0x00000813,
	0x00000833, 0x00000854, 0x0000085d, 0x00000884,
	// Entry 40 - 5F
	0x000008a2, 0x000008b7, 0x000008cc, 0x000008df,
	0x000008fe, 0x00000920, 0x00000927, 0x00000961,
	0x00000996, 0x000009c5, 0x000009dd, 0x00000a01,
	0x00000a5b, 0x00000a9e, 0x00000acd, 0x00000af7,
	0x00000b24, 0x00000b33, 0x00000b3a, 0x00000b3f,
	0x00000b7b, 0x00000b96, 0x00000bb5, 0x00000bd8,
	0x00000bec, 0x00000c43, 0x00000c72, 0x00000c85,
	0x00000c94, 0x00000c9f, 0x00000ccd, 0x00000ce0,
	// Entry 60 - 7F
	0x00000d02, 0x00000d35, 0x00000d6c, 0x00000d8a,
	0x00000d9e, 0x00000db3, 0x00000dd5, 0x00000df3,
	0x00000e12, 0x00000e3b, 0x00000e57, 0x00000eb3,
	0x00000f07, 0x00000f1a, 0x00000f26, 0x00000f46,
	0x00000f4d, 0x00000f54, 0x00000f6d, 0x00000f84,
	0x00000f91, 0x00000fa0, 0x00000fba, 0x00000fca,
	0x00000fe8, 0x00001000, 0x0000100d, 0x00001026,
	0x0000103d, 0x0000104a, 0x00001051, 0x00001058,
	// Entry 80 - 9F
	0x0000105f, 0x00001062, 0x0000106a, 0x00001072,
	0x00001075, 0x00001078, 0x0000107b, 0x0000107e,
} // Size: 568 bytes

const ruData string = "" + // Size: 4222 bytes
	"\x02Установить хост\x02Установить порт\x02<путь>  Установить каталог заг" +
	"рузки при добавлении торрента\x02<имя1,имя2,...>  Установить категории " +
	"при добавлении торрента\x02<имя_файла или URL>  Добавить торрент\x02<0," +
	"1,2,3,...> Отметить файлы для загрузки по номерам индексов\x02Установить" +
	" имя пользователя\x02Установить пароль\x02Показывать полные имена статус" +
	"ов\x02Стартовать добавленный торрент\x02Показывать диалог при добавлени" +
	"и нового торрент-файла (не url/magnet)\x02Вывести адреса трекеров торре" +
	"нт-файла в стандартный вывод\x02Установить интервал обновления информац" +
	"ии о торрентах в секундах\x02Показать версию\x02Все\x02По умолчанию\x02" +
	"Остановлен\x02Ждёт проверки\x02Проверяется\x02В очереди\x02Загрузка\x02" +
	"Раздаётся\x02С ошибкой\x02Статус\x02Готово\x02Время\x02|   Отдача    | " +
	"  Загрузка  | Пиры  | Готово |  Размер   |\x04\x03   \x01 \x07\x02Имя" +
	"\x02Выход\x02Категория\x02Общие\x02Трекеры\x02Пиры\x02Поиск\x02Файлы\x02" +
	"Переместить\x02Помощь\x02Сортировка\x02Для поддержки категорий требуетс" +
	"я transmission-daemon версии 3.00 или больше.\x02Закрыть\x02Сортировка" +
	"\x02Сортировать по\x04\x03   \x00\x1e\x02Дата добавления\x04\x03   \x00" +
	"\x07\x02Имя\x04\x03   \x00\x11\x02Прогресс\x04\x03   \x00\x0d\x02Размер" +
	"\x02Свободно\x02Введите имя новой категории(й)\x02Введите новый путь\x02" +
	"Отмена\x04\x01 \x00\x09\x02Путь\x04\x01 \x01 \x14\x02Категория:\x02Доба" +
	"вить торрент\x04\x01 \x00%\x02Стартовать торрент:\x02нет\x02да\x04\x01 " +
	"\x00\x0d\x02Размер\x02Пробел\x02Получить\x02Свернуть каталог\x02Стартова" +
	"ть да/нет\x02Путь\x02Торрент уже добавлен\x02Горячие клавиши\x02стартов" +
	"ать\x02остановить\x02проверить\x02реаннонсировать\x02удалить торрент(ы)" +
	"\x02или\x02удалить торрент(ы) с содержимым\x02предпросмотр/открыть файл(" +
	"ы)\x02выделить/снять выделение\x02выделить все\x02отменить выделение" +
	"\x02создать новую категорию для выбранных торрентов\x02открыть url из ко" +
	"мментария к торренту\x02открыть каталог загрузки\x02переименовать торре" +
	"нт\x04\x02  \x01 &\x02Готово|  Размер |  Имя\x02Открыть\x02Нет\x02Да" +
	"\x02Вы действительно хотите удалить\x02Переместить в:\x02Переименовать в" +
	":\x02Введите URL трекера:\x02URL трекера:\x02Установить категорию для вы" +
	"деленных торрентов\x02Фильтровать по категории\x02Категории\x02Активны" +
	"\x02Адрес\x04\x02  \x01 '\x02| Пиры  | Сиды  | Статус\x02Приоритет\x02Сл" +
	"едующий каталог\x02Следующий корневой каталог\x04\x00\x01 2\x02|   Разм" +
	"ер  |  Приоритет |  Имя\x02Выбрать каталог\x02Новый путь\x02Директории" +
	"\x02Выбрать категорию\x02Новая категория\x02Редактировать URL\x02Добавит" +
	"ь новый трекер\x02Удалить трекер\x04\x01 \x00W\x02| Готово |  Загрузка " +
	"  |  Отдача   |   Флаги   | Клиент\x02Приостановить/возобновить обновлен" +
	"ия списка\x02Следующий\x02Поиск:\x02Общая информация\x02Имя\x02Хэш\x02Р" +
	"асположение\x02Комментарий\x02Отдано\x02Рейтинг\x02Дата создания\x02Соз" +
	"дан в\x02Дата добавления\x02Общий размер\x02Ошибки\x02Возобновлены\x02О" +
	"становлены\x02Отдача\x02ГиБ\x02МиБ\x02КиБ\x02Б\x02МБ/с\x02кБ/с\x02д\x02" +
	"ч\x02м\x02с"

	// Total table size 7327 bytes (7KiB); checksum: 5334D3F3
