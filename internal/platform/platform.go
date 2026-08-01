package platform

import "fmt"

// Platform describes one retro gaming platform.
type Platform struct {
	ID         int
	Slug       string
	Name       string
	SearchTerm string
}

// All is the full set of supported retro platforms, ordered by IGDB ID.
var All = []Platform{
	{ID: 4, Slug: "n64", Name: "Nintendo 64", SearchTerm: "n64"},
	{ID: 5, Slug: "wii", Name: "Wii", SearchTerm: "wii"},
	{ID: 7, Slug: "ps1", Name: "PlayStation", SearchTerm: "ps1"},
	{ID: 8, Slug: "ps2", Name: "PlayStation 2", SearchTerm: "ps2"},
	{ID: 9, Slug: "ps3", Name: "PlayStation 3", SearchTerm: "ps3"},
	{ID: 11, Slug: "xbox", Name: "Xbox", SearchTerm: "xbox"},
	{ID: 12, Slug: "xbox360", Name: "Xbox 360", SearchTerm: "xbox 360"},
	{ID: 13, Slug: "dos", Name: "PC DOS", SearchTerm: "dos"},
	{ID: 14, Slug: "mac", Name: "Mac", SearchTerm: "mac"},
	{ID: 15, Slug: "c64", Name: "Commodore C64/128", SearchTerm: "commodore 64"},
	{ID: 16, Slug: "amiga", Name: "Amiga", SearchTerm: "amiga"},
	{ID: 18, Slug: "nes", Name: "Nintendo Entertainment System (NES)", SearchTerm: "nes"},
	{ID: 19, Slug: "snes", Name: "Super Nintendo Entertainment System (SNES)", SearchTerm: "snes"},
	{ID: 20, Slug: "nds", Name: "Nintendo DS", SearchTerm: "nintendo ds"},
	{ID: 21, Slug: "gamecube", Name: "Nintendo GameCube", SearchTerm: "gamecube"},
	{ID: 22, Slug: "gbc", Name: "Game Boy Color", SearchTerm: "game boy color"},
	{ID: 23, Slug: "dreamcast", Name: "Dreamcast", SearchTerm: "dreamcast"},
	{ID: 24, Slug: "gba", Name: "Game Boy Advance", SearchTerm: "game boy advance"},
	{ID: 25, Slug: "amstrad-cpc", Name: "Amstrad CPC", SearchTerm: "amstrad cpc"},
	{ID: 26, Slug: "zx-spectrum", Name: "ZX Spectrum", SearchTerm: "zx spectrum"},
	{ID: 27, Slug: "msx", Name: "MSX", SearchTerm: "msx"},
	{ID: 29, Slug: "genesis", Name: "Sega Mega Drive/Genesis", SearchTerm: "sega genesis"},
	{ID: 30, Slug: "sega-32x", Name: "Sega 32X", SearchTerm: "sega 32x"},
	{ID: 32, Slug: "saturn", Name: "Sega Saturn", SearchTerm: "sega saturn"},
	{ID: 33, Slug: "gameboy", Name: "Game Boy", SearchTerm: "game boy"},
	{ID: 35, Slug: "game-gear", Name: "Sega Game Gear", SearchTerm: "sega game gear"},
	{ID: 38, Slug: "psp", Name: "PlayStation Portable", SearchTerm: "psp"},
	{ID: 42, Slug: "n-gage", Name: "N-Gage", SearchTerm: "n-gage"},
	{ID: 44, Slug: "tapwave-zodiac", Name: "Tapwave Zodiac", SearchTerm: "tapwave zodiac"},
	{ID: 50, Slug: "3do", Name: "3DO Interactive Multiplayer", SearchTerm: "3do"},
	{ID: 51, Slug: "famicom-disk-system", Name: "Family Computer Disk System", SearchTerm: "famicom disk system"},
	{ID: 52, Slug: "arcade", Name: "Arcade", SearchTerm: "arcade"},
	{ID: 53, Slug: "msx2", Name: "MSX2", SearchTerm: "msx2"},
	{ID: 57, Slug: "wonderswan", Name: "WonderSwan", SearchTerm: "wonderswan"},
	{ID: 58, Slug: "super-famicom", Name: "Super Famicom", SearchTerm: "super famicom"},
	{ID: 59, Slug: "atari-2600", Name: "Atari 2600", SearchTerm: "atari 2600"},
	{ID: 60, Slug: "atari-7800", Name: "Atari 7800", SearchTerm: "atari 7800"},
	{ID: 61, Slug: "atari-lynx", Name: "Atari Lynx", SearchTerm: "atari lynx"},
	{ID: 62, Slug: "atari-jaguar", Name: "Atari Jaguar", SearchTerm: "atari jaguar"},
	{ID: 63, Slug: "atari-st", Name: "Atari ST/STE", SearchTerm: "atari st"},
	{ID: 64, Slug: "sms", Name: "Sega Master System", SearchTerm: "sega master system"},
	{ID: 65, Slug: "atari-8bit", Name: "Atari 8-bit", SearchTerm: "atari 8-bit"},
	{ID: 66, Slug: "atari-5200", Name: "Atari 5200", SearchTerm: "atari 5200"},
	{ID: 67, Slug: "intellivision", Name: "Intellivision", SearchTerm: "intellivision"},
	{ID: 68, Slug: "colecovision", Name: "ColecoVision", SearchTerm: "colecovision"},
	{ID: 69, Slug: "bbc-micro", Name: "BBC Microcomputer System", SearchTerm: "bbc micro"},
	{ID: 70, Slug: "vectrex", Name: "Vectrex", SearchTerm: "vectrex"},
	{ID: 71, Slug: "vic-20", Name: "Commodore VIC-20", SearchTerm: "commodore vic-20"},
	{ID: 75, Slug: "apple-ii", Name: "Apple II", SearchTerm: "apple ii"},
	{ID: 77, Slug: "sharp-x1", Name: "Sharp X1", SearchTerm: "sharp x1"},
	{ID: 78, Slug: "sega-cd", Name: "Sega CD", SearchTerm: "sega cd"},
	{ID: 79, Slug: "neo-geo-mvs", Name: "Neo Geo MVS", SearchTerm: "neo geo mvs"},
	{ID: 80, Slug: "neo-geo-aes", Name: "Neo Geo AES", SearchTerm: "neo geo aes"},
	{ID: 84, Slug: "sg-1000", Name: "SG-1000", SearchTerm: "sg-1000"},
	{ID: 86, Slug: "turbografx-16", Name: "TurboGrafx-16/PC Engine", SearchTerm: "turbografx-16"},
	{ID: 87, Slug: "virtual-boy", Name: "Virtual Boy", SearchTerm: "virtual boy"},
	{ID: 88, Slug: "odyssey", Name: "Odyssey", SearchTerm: "magnavox odyssey"},
	{ID: 89, Slug: "microvision", Name: "Microvision", SearchTerm: "microvision"},
	{ID: 90, Slug: "pet", Name: "Commodore PET", SearchTerm: "commodore pet"},
	{ID: 91, Slug: "astrocade", Name: "Bally Astrocade", SearchTerm: "bally astrocade"},
	{ID: 93, Slug: "c16", Name: "Commodore 16", SearchTerm: "commodore 16"},
	{ID: 94, Slug: "plus4", Name: "Commodore Plus/4", SearchTerm: "commodore plus 4"},
	{ID: 99, Slug: "famicom", Name: "Family Computer (FAMICOM)", SearchTerm: "famicom"},
	{ID: 114, Slug: "amiga-cd32", Name: "Amiga CD32", SearchTerm: "amiga cd32"},
	{ID: 115, Slug: "apple-iigs", Name: "Apple IIGS", SearchTerm: "apple iigs"},
	{ID: 116, Slug: "archimedes", Name: "Acorn Archimedes", SearchTerm: "acorn archimedes"},
	{ID: 117, Slug: "cdi", Name: "Philips CD-i", SearchTerm: "philips cd-i"},
	{ID: 118, Slug: "fm-towns", Name: "FM Towns", SearchTerm: "fm towns"},
	{ID: 119, Slug: "neo-geo-pocket", Name: "Neo Geo Pocket", SearchTerm: "neo geo pocket"},
	{ID: 120, Slug: "neo-geo-pocket-color", Name: "Neo Geo Pocket Color", SearchTerm: "neo geo pocket color"},
	{ID: 121, Slug: "x68000", Name: "Sharp X68000", SearchTerm: "sharp x68000"},
	{ID: 122, Slug: "nuon", Name: "Nuon", SearchTerm: "nuon"},
	{ID: 123, Slug: "wonderswan-color", Name: "WonderSwan Color", SearchTerm: "wonderswan color"},
	{ID: 124, Slug: "swancrystal", Name: "SwanCrystal", SearchTerm: "swancrystal"},
	{ID: 125, Slug: "pc-8801", Name: "PC-8801", SearchTerm: "pc-8801"},
	{ID: 126, Slug: "trs-80", Name: "TRS-80", SearchTerm: "trs-80"},
	{ID: 127, Slug: "channel-f", Name: "Fairchild Channel F", SearchTerm: "fairchild channel f"},
	{ID: 128, Slug: "supergrafx", Name: "PC Engine SuperGrafx", SearchTerm: "pc engine supergrafx"},
	{ID: 129, Slug: "ti-99", Name: "Texas Instruments TI-99", SearchTerm: "ti-99"},
	{ID: 133, Slug: "videopac", Name: "Philips Videopac G7000", SearchTerm: "philips videopac"},
	{ID: 134, Slug: "acorn-electron", Name: "Acorn Electron", SearchTerm: "acorn electron"},
	{ID: 135, Slug: "hyper-neo-geo-64", Name: "Hyper Neo Geo 64", SearchTerm: "hyper neo geo 64"},
	{ID: 136, Slug: "neo-geo-cd", Name: "Neo Geo CD", SearchTerm: "neo geo cd"},
	{ID: 138, Slug: "vc-4000", Name: "VC 4000", SearchTerm: "vc 4000"},
	{ID: 139, Slug: "1292-apvs", Name: "1292 Advanced Programmable Video System", SearchTerm: "1292 advanced programmable video system"},
	{ID: 149, Slug: "pc-98", Name: "PC-98", SearchTerm: "pc-98"},
	{ID: 150, Slug: "turbografx-cd", Name: "Turbografx-16/PC Engine CD", SearchTerm: "turbografx cd"},
	{ID: 151, Slug: "trs-80-coco", Name: "TRS-80 Color Computer", SearchTerm: "trs-80 color computer"},
	{ID: 152, Slug: "fm-7", Name: "FM-7", SearchTerm: "fm-7"},
	{ID: 153, Slug: "dragon", Name: "Dragon 32/64", SearchTerm: "dragon 32"},
	{ID: 154, Slug: "amstrad-pcw", Name: "Amstrad PCW", SearchTerm: "amstrad pcw"},
	{ID: 155, Slug: "tatung-einstein", Name: "Tatung Einstein", SearchTerm: "tatung einstein"},
	{ID: 156, Slug: "thomson-mo5", Name: "Thomson MO5", SearchTerm: "thomson mo5"},
	{ID: 157, Slug: "pc-6000", Name: "NEC PC-6000 Series", SearchTerm: "nec pc-6000"},
	{ID: 158, Slug: "cdtv", Name: "Commodore CDTV", SearchTerm: "commodore cdtv"},
	{ID: 159, Slug: "dsi", Name: "Nintendo DSi", SearchTerm: "nintendo dsi"},
	{ID: 166, Slug: "pokemon-mini", Name: "Pokémon mini", SearchTerm: "pokemon mini"},
}

// DefaultSlug is the platform used when --platform is not specified.
const DefaultSlug = "gamecube"

// BySlug looks up a platform by its CLI-facing slug (e.g. "gamecube", "snes").
func BySlug(slug string) (Platform, error) {
	for _, p := range All {
		if p.Slug == slug {
			return p, nil
		}
	}
	return Platform{}, fmt.Errorf("unknown platform %q (see `ingestor platforms` for the full list)", slug)
}

// ByName looks up a platform by its canonical name.
func ByName(name string) (Platform, error) {
	for _, p := range All {
		if p.Name == name {
			return p, nil
		}
	}
	return Platform{}, fmt.Errorf("unrecognized platform name %q", name)
}
