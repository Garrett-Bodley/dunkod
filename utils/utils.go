package utils

import (
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"runtime"
	"time"
	"unicode"

	"dunkod/config"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func ErrorWithTrace(e error) error {
	_, file, line, _ := runtime.Caller(1)
	return fmt.Errorf("%s:%d\n\t%v", file, line, e)
}

func IsInvalidSeason(season string) bool {
	for _, s := range config.ValidSeasons {
		if season == s {
			return false
		}
	}
	return true
}

var sem = make(chan int, 50)

func CurlToFile(url, filepath string) error {
	sem <- 1
	defer func() { <-sem }()
	client := &http.Client{
		Timeout: time.Minute,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ErrorWithTrace(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return ErrorWithTrace(err)
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return ErrorWithTrace(err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return ErrorWithTrace(err)
	}
	return nil
}

func Curl(req *http.Request) ([]byte, error) {
	sem <- 1
	defer func() { <-sem }()
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrorWithTrace(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrorWithTrace(err)
	}
	return body, nil
}

var activities = []string{
	"Monster-Dunk",
	"Clutch-Three",
	"Logo-Three",
	"Posterize",
	"No-Look",
	"Pump-Fake",
	"Nutmeg",
	"Crossover",
	"Shamgod",
	"Bank-Shot",
	"Reverse-Layup",
	"Sky-Hook",
	"Hesi-Tween",
	"Pull-Up",
	"Rack-Attack",
	"Step-Through",
	"Block",
	"Alley-Oop",
	"Post-Up",
	"And-One",
	"Ball-Dont-Lie",
	"Chase-Down",
	"Finger-Roll",
	"Poster-Dunk",
	"Wet-Jumper",
	"Jab-Step",
	"Buzzer-Beater",
	"Circus-Shot",
	"Coast-To-Coast",
	"Ankle-Breaker",
	"Denied-At-The-Rim",
	"Lockdown-Defense",
	"Rim-Runner",
	"Heat-Check",
	"Floater",
	"Pure-Shooter",
	"Dagger-Shot",
	"Killer-Instinct",
	"Clamps",
	"Clutch-Gene",
	"Wheeling-Dealing",
	"Swishing-Dishing",
	"Styling-Profiling",
	"Vintage",
	"Percolating-Devastating",
	"Razzle-Dazzle",
	"Suffocating-D",
	"Winning-Grinning",
	"Tomahawk-Stuff",
}

var allStars = []string{
	"LeBronJames",
	"KareemAbdul-Jabbar",
	"KobeBryant",
	"TimDuncan",
	"KevinDurant",
	"KevinGarnett",
	"ShaquilleONeal",
	"MichaelJordan",
	"KarlMalone",
	"DirkNowitzki",
	"JerryWest",
	"WiltChamberlain",
	"BobCousy",
	"JohnHavlicek",
	"DwyaneWade",
	"LarryBird",
	"ElvinHayes",
	"MagicJohnson",
	"MosesMalone",
	"HakeemOlajuwon",
	"ChrisPaul",
	"OscarRobertson",
	"BillRussell",
	"DolphSchayes",
	"IsiahThomas",
	"CharlesBarkley",
	"ElginBaylor",
	"ChrisBosh",
	"StephenCurry",
	"JuliusErving",
	"PatrickEwing",
	"JamesHarden",
	"AllenIverson",
	"BobPettit",
	"RayAllen",
	"CarmeloAnthony",
	"PaulArizin",
	"AnthonyDavis",
	"ClydeDrexler",
	"HalGreer",
	"JasonKidd",
	"PaulPierce",
	"DavidRobinson",
	"JohnStockton",
	"GiannisAntetokounmpo",
	"PaulGeorge",
	"GeorgeGervin",
	"KyrieIrving",
	"DamianLillard",
	"RobertParish",
	"GaryPayton",
	"RussellWestbrook",
	"LennyWilkens",
	"DominiqueWilkins",
	"RickBarry",
	"VinceCarter",
	"DaveCowens",
	"DaveDeBusschere",
	"AlexEnglish",
	"LarryFoust",
	"DwightHoward",
	"BobLanier",
	"DikembeMutombo",
	"SteveNash",
	"BillSharman",
	"YaoMing",
	"LaMarcusAldridge",
	"DaveBing",
	"JoelEmbiid",
	"WaltFrazier",
	"HarryGallatin",
	"GrantHill",
	"JoeJohnson",
	"NikolaJokic",
	"JerryLucas",
	"EdMacauley",
	"SlaterMartin",
	"TracyMcGrady",
	"DickMcGuire",
	"KevinMcHale",
	"AlonzoMourning",
	"ScottiePippen",
	"WillisReed",
	"JackSikma",
	"NateThurmond",
	"ChetWalker",
	"JoJoWhite",
	"JamesWorthy",
	"NateArchibald",
	"JimmyButler",
	"LarryCostello",
	"AdrianDantley",
	"WalterDavis",
	"DeMarDeRozan",
	"JoeDumars",
	"PauGasol",
	"ArtisGilmore",
	"BlakeGriffin",
	"RichieGuerin",
	"TomHeinsohn",
	"BaileyHowell",
	"LouHudson",
	"NeilJohnston",
	"ShawnKemp",
	"KawhiLeonard",
	"KyleLowry",
	"VernMikkelsen",
	"DonovanMitchell",
	"JermaineONeal",
	"TonyParker",
	"MitchRichmond",
	"AmareStoudemire",
	"JaysonTatum",
	"JackTwyman",
	"GeorgeYardley",
	"ChaunceyBillups",
	"CarlBraun",
	"BradDaugherty",
	"LukaDoncic",
	"WayneEmbry",
	"TomGola",
	"GailGoodrich",
	"CliffHagan",
	"TimHardaway",
	"AlHorford",
	"DennisJohnson",
	"GusJohnson",
	"MarquesJohnson",
	"SamJones",
	"RudyLaRusso",
	"KevinLove",
	"PeteMaravich",
	"BobMcAdoo",
	"ReggieMiller",
	"SidneyMoncrief",
	"ChrisMullin",
	"DonOhl",
	"AndyPhillip",
	"GeneShue",
	"KlayThompson",
	"RudyTomjanovich",
	"Karl-AnthonyTowns",
	"WesUnseld",
	"JohnWall",
	"BobbyWanzer",
	"ChrisWebber",
	"PaulWestphal",
	"VinBaker",
	"WaltBellamy",
	"OtisBirdsong",
	"RolandoBlackman",
	"DevinBooker",
	"JaylenBrown",
	"TomChambers",
	"MauriceCheeks",
	"DougCollins",
	"DeMarcusCousins",
	"BillyCunningham",
	"BobDandridge",
	"BobDavies",
	"DickGarmaker",
	"DraymondGreen",
	"JohnnyGreen",
	"PennyHardaway",
	"ConnieHawkins",
	"SpencerHaywood",
	"MelHutchins",
	"BobbyJones",
	"BernardKing",
	"BillLaimbeer",
	"ClydeLovellette",
	"MauriceLucas",
	"ShawnMarion",
	"GeorgeMikan",
	"PaulMillsap",
	"EarlMonroe",
	"WillieNaulls",
	"JimPollard",
	"MarkPrice",
	"MichealRayRichardson",
	"ArnieRisen",
	"AlvinRobertson",
	"GuyRodgers",
	"RajonRondo",
	"RalphSampson",
	"LatrellSprewell",
	"DavidThompson",
	"KembaWalker",
	"BenWallace",
	"RasheedWallace",
	"SidneyWicks",
	"TraeYoung",
	"BamAdebayo",
	"MarkAguirre",
	"GilbertArenas",
	"BradleyBeal",
	"BillBridges",
	"PhilChenier",
	"TerryDischinger",
	"AnthonyEdwards",
	"SteveFrancis",
	"MarcGasol",
	"ShaiGilgeous-Alexander",
	"RudyGobert",
	"RichardHamilton",
	"KevinJohnson",
	"EddieJones",
	"BobKauffman",
	"JohnnyKerr",
	"BobLove",
	"DanMajerle",
	"GeorgeMcGinnis",
	"KhrisMiddleton",
	"JeffMullins",
	"LarryNance",
	"JuliusRandle",
	"GlenRice",
	"DerrickRose",
	"DanRoundfield",
	"BrandonRoy",
	"DomantasSabonis",
	"DetlefSchrempf",
	"CharlieScott",
	"PaulSeymour",
	"PascalSiakam",
	"BenSimmons",
	"PejaStojakovic",
	"MauriceStokes",
	"DickVanArsdale",
	"TomVanArsdale",
	"NormVanLier",
	"AntoineWalker",
	"JamaalWilkes",
	"BuckWilliams",
	"DeronWilliams",
	"LeoBarnhorst",
	"ZelmoBeaty",
	"CarlosBoozer",
	"EltonBrand",
	"TerrellBrandon",
	"FrankBrian",
	"JalenBrunson",
	"CaronButler",
	"JoeCaldwell",
	"ArchieClark",
	"TerryCummings",
	"BaronDavis",
	"LuolDeng",
	"JohnDrew",
	"AndreDrummond",
	"KevinDuckworth",
	"WalterDukes",
	"DwightEddleman",
	"SeanElliott",
	"MichaelFinley",
	"JoeFulks",
	"DariusGarland",
	"JackGeorge",
	"ManuGinobili",
	"TyreseHaliburton",
	"RoyHibbert",
	"JrueHoliday",
	"AllanHouston",
	"RodHundley",
	"ZydrunasIlgauskas",
	"JarenJacksonJr",
	"AntawnJamison",
	"EddieJohnson",
	"JohnJohnson",
	"LarryJohnson",
	"LarryKenon",
	"DonKojis",
	"ZachLaVine",
	"DavidLee",
	"FatLever",
	"RashardLewis",
	"JeffMalone",
	"DannyManning",
	"StephonMarbury",
	"JackMarin",
	"BradMiller",
	"JaMorant",
	"NormNixon",
	"JoakimNoah",
	"VictorOladipo",
	"JimPaxson",
	"GeoffPetrie",
	"TerryPorter",
	"ZachRandolph",
	"GlennRobinson",
	"TruckRobinson",
	"RedRocha",
	"DennisRodman",
	"JeffRuland",
	"FredScolari",
	"KenSears",
	"FrankSelvy",
	"PaulSilas",
	"JerrySloan",
	"PhilSmith",
	"RandySmith",
	"JerryStackhouse",
	"ReggieTheus",
	"IsaiahThomas",
	"AndrewToney",
	"KellyTripucka",
	"KikiVandeweghe",
	"NikolaVucevic",
	"JimmyWalker",
	"BillWalton",
	"ScottWedman",
	"DavidWest",
	"GusWilliams",
	"ZionWilliamson",
	"BrianWinters",
	"ShareefAbdur-Rahim",
	"AlvanAdams",
	"MichaelAdams",
	"DannyAinge",
	"JarrettAllen",
	"KennyAnderson",
	"BJArmstrong",
	"LaMeloBall",
	"PaoloBanchero",
	"DonBarksdale",
	"ScottieBarnes",
	"DickBarnett",
	"DanaBarros",
	"ButchBeard",
	"RalphBeard",
	"MookieBlaylock",
	"JohnBlock",
	"BobBoozer",
	"VinceBoryla",
	"BillBradley",
	"FredBrown",
	"DonBuse",
	"AndrewBynum",
	"AustinCarr",
	"JoeBarryCarroll",
	"BillCartwright",
	"SamCassell",
	"CedricCeballos",
	"TysonChandler",
	"LenChappell",
	"NathanielClifton",
	"DerrickColeman",
	"JackColeman",
	"MikeConley",
	"CadeCunningham",
	"AntonioDavis",
	"DaleDavis",
	"VladeDivac",
	"JamesDonaldson",
	"GoranDragic",
	"MarkEaton",
	"DaleEllis",
	"RayFelix",
	"SleepyFloyd",
	"DeAaronFox",
	"WorldBFree",
	"BillyGabor",
	"ChrisGatling",
	"DannyGranger",
	"HoraceGrant",
	"ACGreen",
	"RickeyGreen",
	"AlexGroza",
	"TomGugliotta",
	"DevinHarris",
	"BobHarrison",
	"HerseyHawkins",
	"GordonHayward",
	"WaltHazzard",
	"TylerHerro",
	"TyroneHill",
	"LionelHollins",
	"JeffHornacek",
	"JoshHoward",
	"JuwanHoward",
	"AndreIguodala",
	"DarrallImhoff",
	"BrandonIngram",
	"DanIssel",
	"LuciousJackson",
	"MarkJackson",
	"SteveJohnson",
	"DeAndreJordan",
	"ChrisKaman",
	"JimKing",
	"AndreiKirilenko",
	"BillyKnight",
	"KyleKorver",
	"SamLacey",
	"ChristianLaettner",
	"ClydeLee",
	"ReggieLewis",
	"BrookLopez",
	"JamaalMagloire",
	"LauriMarkkanen",
	"KenyonMartin",
	"JamalMashburn",
	"AnthonyMason",
	"TyreseMaxey",
	"XavierMcDaniel",
	"AntonioMcDyess",
	"JonMcGlocklin",
	"TomMeschery",
	"EddieMiles",
	"MikeMitchell",
	"SteveMix",
	"EvanMobley",
	"JackMolinas",
	"CalvinMurphy",
	"DejounteMurray",
	"CalvinNatt",
	"JameerNelson",
	"ChuckNoble",
	"CharlesOakley",
	"MehmetOkur",
	"RickyPierce",
	"KristapsPorzingis",
	"JimPrice",
	"TheoRatliff",
	"MichaelRedd",
	"RichieRegan",
	"DocRivers",
	"CliffordRobinson",
	"FlynnRobinson",
	"CurtisRowe",
	"BobRule",
	"CampyRussell",
	"CazzieRussell",
	"DAngeloRussell",
	"WoodySauldsberry",
	"FredSchaus",
	"AlperenSengun",
	"LeeShaffer",
	"LonnieShelton",
	"AdrianSmith",
	"SteveSmith",
	"RikSmits",
	"JohnStarks",
	"DonSunderlage",
	"WallySzczerbiak",
	"JeffTeague",
	"OtisThorpe",
	"NickVanExel",
	"FredVanVleet",
	"GeraldWallace",
	"PaulWalther",
	"KermitWashington",
	"VictorWembanyama",
	"AndrewWiggins",
	"JalenWilliams",
	"JaysonWilliams",
	"MoWilliams",
	"KevinWillis",
	"MettaWorldPeace",
	"MaxZaslofsky",
}

func getActivity() string {
	n := len(activities)
	return activities[rand.IntN(n)]
}

func getAllStar() string {
	n := len(allStars)
	return allStars[rand.IntN(n)]
}

func CreateSlug() string {
	name1 := getAllStar()
	name2 := getAllStar()
	for name1 == name2 {
		name2 = getAllStar()
	}
	activity := getActivity()
	if activity == "Vintage" {
		return activity + "-" + name1 + "-" + name2
	}
	return name1 + "-" + name2 + "-" + activity
}

func RemoveDiacritics(input string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	res, _, err := transform.String(t, input)
	if err != nil {
		return input
	}
	return res
}
