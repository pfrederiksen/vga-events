package course

import "strings"

// manualDatabase contains information for ~50 famous golf courses
// This is used as a fallback when API lookups fail or are unavailable
var manualDatabase = map[string]*CourseInfo{
	// California - Famous Courses
	"pebble beach": {
		Name:    "Pebble Beach Golf Links",
		City:    "Pebble Beach",
		State:   "CA",
		Yardage: 6828,
		Par:     72,
		Slope:   145,
		Rating:  74.5,
		Website: "https://www.pebblebeach.com",
		Source:  "manual",
	},
	"spyglass hill": {
		Name:    "Spyglass Hill Golf Course",
		City:    "Pebble Beach",
		State:   "CA",
		Yardage: 6953,
		Par:     72,
		Slope:   147,
		Rating:  75.5,
		Website: "https://www.pebblebeach.com",
		Source:  "manual",
	},
	"torrey pines": {
		Name:    "Torrey Pines Golf Course (South)",
		City:    "La Jolla",
		State:   "CA",
		Yardage: 7698,
		Par:     72,
		Slope:   141,
		Rating:  77.0,
		Website: "https://www.sandiego.gov/park-and-recreation/golf/torreypines",
		Source:  "manual",
	},
	"riviera": {
		Name:    "Riviera Country Club",
		City:    "Pacific Palisades",
		State:   "CA",
		Yardage: 7322,
		Par:     71,
		Slope:   139,
		Rating:  75.3,
		Website: "https://www.therivieracountryclub.com",
		Source:  "manual",
	},
	"olympic club": {
		Name:    "Olympic Club (Lake Course)",
		City:    "San Francisco",
		State:   "CA",
		Yardage: 7169,
		Par:     71,
		Slope:   142,
		Rating:  75.4,
		Website: "https://www.olyclub.com",
		Source:  "manual",
	},

	// Nevada - Famous Courses
	"shadow creek": {
		Name:    "Shadow Creek Golf Course",
		City:    "Las Vegas",
		State:   "NV",
		Yardage: 7560,
		Par:     72,
		Slope:   144,
		Rating:  76.5,
		Website: "https://www.mgmresorts.com/en/things-to-do/golf/shadow-creek.html",
		Source:  "manual",
	},
	"cascata": {
		Name:    "Cascata Golf Club",
		City:    "Boulder City",
		State:   "NV",
		Yardage: 7137,
		Par:     72,
		Slope:   146,
		Rating:  74.6,
		Website: "https://www.cascatagolf.com",
		Source:  "manual",
	},
	"wolf creek": {
		Name:    "Wolf Creek Golf Club",
		City:    "Mesquite",
		State:   "NV",
		Yardage: 6939,
		Par:     72,
		Slope:   143,
		Rating:  73.8,
		Website: "https://www.golfwolfcreek.com",
		Source:  "manual",
	},

	// Arizona - Famous Courses
	"we-ko-pa": {
		Name:    "We-Ko-Pa Golf Club (Saguaro)",
		City:    "Fort McDowell",
		State:   "AZ",
		Yardage: 7225,
		Par:     72,
		Slope:   137,
		Rating:  74.3,
		Website: "https://www.wekopa.com",
		Source:  "manual",
	},
	"troon north": {
		Name:    "Troon North Golf Club (Monument)",
		City:    "Scottsdale",
		State:   "AZ",
		Yardage: 7070,
		Par:     72,
		Slope:   147,
		Rating:  74.3,
		Website: "https://www.troonnorthgolf.com",
		Source:  "manual",
	},
	"grayhawk": {
		Name:    "Grayhawk Golf Club (Raptor)",
		City:    "Scottsdale",
		State:   "AZ",
		Yardage: 7135,
		Par:     72,
		Slope:   142,
		Rating:  74.1,
		Website: "https://www.grayhawkgolf.com",
		Source:  "manual",
	},

	// Texas - Famous Courses
	"colonial": {
		Name:    "Colonial Country Club",
		City:    "Fort Worth",
		State:   "TX",
		Yardage: 7209,
		Par:     70,
		Slope:   139,
		Rating:  75.4,
		Website: "https://www.colonialfw.com",
		Source:  "manual",
	},
	"trinity forest": {
		Name:    "Trinity Forest Golf Club",
		City:    "Dallas",
		State:   "TX",
		Yardage: 7350,
		Par:     71,
		Slope:   141,
		Rating:  75.9,
		Website: "https://www.trinityforestgolf.com",
		Source:  "manual",
	},

	// Florida - Famous Courses
	"tpc sawgrass": {
		Name:    "TPC Sawgrass (Stadium Course)",
		City:    "Ponte Vedra Beach",
		State:   "FL",
		Yardage: 7256,
		Par:     72,
		Slope:   155,
		Rating:  76.8,
		Website: "https://www.tpc.com/sawgrass",
		Source:  "manual",
	},
	"bay hill": {
		Name:    "Bay Hill Club & Lodge",
		City:    "Orlando",
		State:   "FL",
		Yardage: 7419,
		Par:     72,
		Slope:   145,
		Rating:  77.1,
		Website: "https://www.bayhill.com",
		Source:  "manual",
	},
	"seminole": {
		Name:    "Seminole Golf Club",
		City:    "Juno Beach",
		State:   "FL",
		Yardage: 6903,
		Par:     72,
		Slope:   142,
		Rating:  73.6,
		Website: "https://www.seminolegolfclub.org",
		Source:  "manual",
	},

	// Georgia - Famous Courses
	"augusta national": {
		Name:    "Augusta National Golf Club",
		City:    "Augusta",
		State:   "GA",
		Yardage: 7510,
		Par:     72,
		Slope:   155,
		Rating:  78.1,
		Website: "https://www.masters.com",
		Source:  "manual",
	},
	"east lake": {
		Name:    "East Lake Golf Club",
		City:    "Atlanta",
		State:   "GA",
		Yardage: 7346,
		Par:     70,
		Slope:   143,
		Rating:  76.4,
		Website: "https://www.eastlakegolfclub.com",
		Source:  "manual",
	},

	// North Carolina - Famous Courses
	"pinehurst no. 2": {
		Name:    "Pinehurst No. 2",
		City:    "Pinehurst",
		State:   "NC",
		Yardage: 7588,
		Par:     72,
		Slope:   155,
		Rating:  77.5,
		Website: "https://www.pinehurst.com",
		Source:  "manual",
	},
	"pinehurst no. 4": {
		Name:    "Pinehurst No. 4",
		City:    "Pinehurst",
		State:   "NC",
		Yardage: 7135,
		Par:     72,
		Slope:   141,
		Rating:  74.8,
		Website: "https://www.pinehurst.com",
		Source:  "manual",
	},

	// South Carolina - Famous Courses
	"kiawah island": {
		Name:    "Kiawah Island Golf Resort (Ocean Course)",
		City:    "Kiawah Island",
		State:   "SC",
		Yardage: 7876,
		Par:     72,
		Slope:   155,
		Rating:  79.6,
		Website: "https://www.kiawahresort.com",
		Source:  "manual",
	},
	"harbour town": {
		Name:    "Harbour Town Golf Links",
		City:    "Hilton Head Island",
		State:   "SC",
		Yardage: 7188,
		Par:     71,
		Slope:   145,
		Rating:  75.1,
		Website: "https://www.seapines.com/golf/harbour-town-golf-links",
		Source:  "manual",
	},

	// New York - Famous Courses
	"bethpage black": {
		Name:    "Bethpage State Park (Black Course)",
		City:    "Farmingdale",
		State:   "NY",
		Yardage: 7468,
		Par:     71,
		Slope:   155,
		Rating:  77.5,
		Website: "https://www.nysparks.com/golf-courses/27/details.aspx",
		Source:  "manual",
	},
	"shinnecock hills": {
		Name:    "Shinnecock Hills Golf Club",
		City:    "Southampton",
		State:   "NY",
		Yardage: 7445,
		Par:     70,
		Slope:   155,
		Rating:  78.5,
		Website: "https://www.shinnecockhills.com",
		Source:  "manual",
	},

	// Pennsylvania - Famous Courses
	"oakmont": {
		Name:    "Oakmont Country Club",
		City:    "Oakmont",
		State:   "PA",
		Yardage: 7255,
		Par:     71,
		Slope:   155,
		Rating:  77.3,
		Website: "https://www.oakmontcountryclub.com",
		Source:  "manual",
	},
	"merion": {
		Name:    "Merion Golf Club (East Course)",
		City:    "Ardmore",
		State:   "PA",
		Yardage: 6996,
		Par:     70,
		Slope:   145,
		Rating:  75.5,
		Website: "https://www.meriongolfclub.com",
		Source:  "manual",
	},

	// New Jersey - Famous Courses
	"pine valley": {
		Name:    "Pine Valley Golf Club",
		City:    "Pine Valley",
		State:   "NJ",
		Yardage: 7057,
		Par:     70,
		Slope:   155,
		Rating:  76.8,
		Website: "https://www.pinevalley.org",
		Source:  "manual",
	},
	"baltusrol": {
		Name:    "Baltusrol Golf Club (Lower Course)",
		City:    "Springfield",
		State:   "NJ",
		Yardage: 7428,
		Par:     72,
		Slope:   146,
		Rating:  76.5,
		Website: "https://www.baltusrol.org",
		Source:  "manual",
	},

	// Michigan - Famous Courses
	"oakland hills": {
		Name:    "Oakland Hills Country Club (South Course)",
		City:    "Bloomfield Hills",
		State:   "MI",
		Yardage: 7395,
		Par:     70,
		Slope:   148,
		Rating:  77.1,
		Website: "https://www.oaklandhillscc.com",
		Source:  "manual",
	},

	// Wisconsin - Famous Courses
	"whistling straits": {
		Name:    "Whistling Straits (Straits Course)",
		City:    "Sheboygan",
		State:   "WI",
		Yardage: 7790,
		Par:     72,
		Slope:   151,
		Rating:  78.1,
		Website: "https://www.americanclubresort.com/golf/whistling-straits",
		Source:  "manual",
	},
	"erin hills": {
		Name:    "Erin Hills Golf Course",
		City:    "Erin",
		State:   "WI",
		Yardage: 7842,
		Par:     72,
		Slope:   152,
		Rating:  78.3,
		Website: "https://www.erinhills.com",
		Source:  "manual",
	},

	// Oregon - Famous Courses
	"bandon dunes": {
		Name:    "Bandon Dunes Golf Resort (Bandon Dunes)",
		City:    "Bandon",
		State:   "OR",
		Yardage: 6732,
		Par:     72,
		Slope:   138,
		Rating:  73.5,
		Website: "https://www.bandondunesgolf.com",
		Source:  "manual",
	},
	"pacific dunes": {
		Name:    "Bandon Dunes Golf Resort (Pacific Dunes)",
		City:    "Bandon",
		State:   "OR",
		Yardage: 6633,
		Par:     71,
		Slope:   138,
		Rating:  73.3,
		Website: "https://www.bandondunesgolf.com",
		Source:  "manual",
	},

	// Washington - Famous Courses
	"chambers bay": {
		Name:    "Chambers Bay Golf Course",
		City:    "University Place",
		State:   "WA",
		Yardage: 7585,
		Par:     72,
		Slope:   145,
		Rating:  76.9,
		Website: "https://www.chambersbaygolf.com",
		Source:  "manual",
	},

	// Colorado - Famous Courses
	"castle pines": {
		Name:    "Castle Pines Golf Club",
		City:    "Castle Rock",
		State:   "CO",
		Yardage: 7559,
		Par:     72,
		Slope:   152,
		Rating:  76.8,
		Website: "https://www.castlepinesgolf.com",
		Source:  "manual",
	},

	// Utah - Famous Courses
	"sand hollow": {
		Name:    "Sand Hollow Resort (Championship)",
		City:    "Hurricane",
		State:   "UT",
		Yardage: 7321,
		Par:     73,
		Slope:   141,
		Rating:  74.9,
		Website: "https://www.sandhollowresort.com",
		Source:  "manual",
	},

	// Hawaii - Famous Courses
	"kapalua": {
		Name:    "Kapalua Resort (Plantation Course)",
		City:    "Lahaina",
		State:   "HI",
		Yardage: 7596,
		Par:     73,
		Slope:   145,
		Rating:  76.4,
		Website: "https://www.golfatkapalua.com",
		Source:  "manual",
	},
	"mauna kea": {
		Name:    "Mauna Kea Golf Course",
		City:    "Kamuela",
		State:   "HI",
		Yardage: 7370,
		Par:     72,
		Slope:   145,
		Rating:  75.4,
		Website: "https://www.maunakeabeachhotel.com/golf",
		Source:  "manual",
	},

	// Illinois - Famous Courses
	"cog hill": {
		Name:    "Cog Hill Golf & Country Club (Dubsdread)",
		City:    "Lemont",
		State:   "IL",
		Yardage: 7366,
		Par:     72,
		Slope:   142,
		Rating:  76.0,
		Website: "https://www.coghillgolf.com",
		Source:  "manual",
	},

	// Minnesota - Famous Courses
	"hazeltine": {
		Name:    "Hazeltine National Golf Club",
		City:    "Chaska",
		State:   "MN",
		Yardage: 7674,
		Par:     72,
		Slope:   148,
		Rating:  77.4,
		Website: "https://www.hazeltinenational.com",
		Source:  "manual",
	},
}

// lookupManual searches the manual database for course information
func lookupManual(title, state string) *CourseInfo {
	normalized := normalizeTitle(title)

	// Direct lookup by normalized title
	if info, exists := manualDatabase[normalized]; exists {
		// Verify state matches if provided
		if state == "" || strings.EqualFold(info.State, state) {
			return info
		}
	}

	// Fallback: try partial matching (substring search)
	for key, info := range manualDatabase {
		if strings.Contains(key, normalized) || strings.Contains(normalized, key) {
			// Verify state matches if provided
			if state == "" || strings.EqualFold(info.State, state) {
				return info
			}
		}
	}

	return nil
}
