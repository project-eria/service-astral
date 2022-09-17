package main

import (
	"fmt"
	"time"

	"github.com/project-eria/go-wot/thing"

	"github.com/go-co-op/gocron"

	eria "github.com/project-eria/eria-core"
	"github.com/project-eria/eria-core/model"
	"github.com/rs/zerolog/log"
	"github.com/sj14/astral"
)

type AstralInfo struct {
	name   string
	desc   string
	getter func(time.Time) time.Time
}

var (
	// Version is a placeholder that will receive the git tag version during build time
	Version      = "-"
	_location    *time.Location
	_thing       *eria.EriaThing
	_scheduler   *gocron.Scheduler
	_observer    astral.Observer
	_astralTimes map[string]AstralInfo
	_next        map[string]*gocron.Job
)

var config = struct {
	Host        string  `yaml:"host"`
	Port        uint    `yaml:"port" default:"80"`
	ExposedAddr string  `yaml:"exposedAddr"`
	Lat         float64 `yaml:"lat" required:"true"`
	Long        float64 `yaml:"long" required:"true"`
	Location    string  `yaml:"location" required:"true"`
}{}

func init() {
	eria.Init("ERIA Ephemeris Info", Version)
	// Loading config
	eria.LoadConfig(&config)
	location, err := time.LoadLocation(config.Location)
	if err != nil {
		log.Error().Err(err).Msg("[init]")
		return
	}
	_location = location
	_observer = astral.Observer{Latitude: config.Lat, Longitude: config.Long, Elevation: 0}
	_astralTimes = map[string]AstralInfo{
		"dawnAstronomical": AstralInfo{
			name: "Dawn (Astronomical)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dawn(_observer, t, astral.DepressionAstronomical)
				if err != nil {
					log.Error().Err(err).Msg("[init:dawnAstronomical]")
				}
				return value
			},
		},
		"dawnNautical": AstralInfo{
			name: "Dawn (Nautical)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dawn(_observer, t, astral.DepressionNautical)
				if err != nil {
					log.Error().Err(err).Msg("[init:dawnNautical]")
				}
				return value
			},
		},
		"dawnCivil": AstralInfo{
			name: "Dawn (Civil)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dawn(_observer, t, astral.DepressionCivil)
				if err != nil {
					log.Error().Err(err).Msg("[init:dawnCivil]")
				}
				return value
			},
		},
		"goldenHourRisingStart": AstralInfo{
			name: "Golden Hour Start (Rising)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, _, err := astral.GoldenHour(_observer, t, astral.SunDirectionRising)
				if err != nil {
					log.Error().Err(err).Msg("[init:goldenHourRisingStart]")
				}
				return value
			},
		},
		"sunrise": AstralInfo{
			name: "Sunrise",
			desc: "the Sun appears on the horizon in the morning",
			getter: func(t time.Time) time.Time {
				value, err := astral.Sunrise(_observer, t)
				if err != nil {
					log.Error().Err(err).Msg("[init:sunrise]")
				}
				return value
			},
		},
		"goldenHourRisingEnd": AstralInfo{
			name: "Golden Hour End (Rising)",
			desc: "",
			getter: func(t time.Time) time.Time {
				_, value, err := astral.GoldenHour(_observer, t, astral.SunDirectionRising)
				if err != nil {
					log.Error().Err(err).Msg("[init:goldenHourRisingEnd]")
				}
				return value
			},
		},
		"noon": AstralInfo{
			name: "noon",
			desc: "",
			getter: func(t time.Time) time.Time {
				value := astral.Noon(_observer, t)
				return value
			},
		},
		"goldenHourSettingStart": AstralInfo{
			name: "Golden Hour Start (Setting)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, _, err := astral.GoldenHour(_observer, t, astral.SunDirectionSetting)
				if err != nil {
					log.Error().Err(err).Msg("[init:goldenHourSettingStart]")
				}
				return value
			},
		},
		"sunset": AstralInfo{
			name: "Sunset",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Sunset(_observer, t)
				if err != nil {
					log.Error().Err(err).Msg("[init:sunset]")
				}
				return value
			},
		},
		"goldenHourSettingEnd": AstralInfo{
			name: "Golden Hour End (Setting)",
			desc: "",
			getter: func(t time.Time) time.Time {
				_, value, err := astral.GoldenHour(_observer, t, astral.SunDirectionSetting)
				if err != nil {
					log.Error().Err(err).Msg("[init:goldenHourSettingEnd]")
				}
				return value
			},
		},
		"duskCivil": AstralInfo{
			name: "Dusk (Civil)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dusk(_observer, t, astral.DepressionCivil)
				if err != nil {
					log.Error().Err(err).Msg("[init:duskCivil]")
				}
				return value
			},
		},
		"duskNautical": AstralInfo{
			name: "Dusk (Nautical)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dusk(_observer, t, astral.DepressionNautical)
				if err != nil {
					log.Error().Err(err).Msg("[init:duskNautical]")
				}
				return value
			},
		},
		"duskAstronomical": AstralInfo{
			name: "Dusk (Astronomical)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dusk(_observer, t, astral.DepressionAstronomical)
				if err != nil {
					log.Error().Err(err).Msg("[init:duskAstronomical]")
				}
				return value
			},
		},
		"midnight": AstralInfo{
			name: "Midnight",
			desc: "",
			getter: func(t time.Time) time.Time {
				value := astral.Midnight(_observer, t)
				return value
			},
		},
	}
	_next = map[string]*gocron.Job{}
}

func main() {
	defer func() {
		log.Info().Msg("[main] Stopped")
	}()

	eriaServer := eria.NewServer(config.Host, config.Port, config.ExposedAddr)

	td, _ := eria.NewThingDescription(
		"eria:service:astral:1",
		Version,
		"Astral",
		"Calculations for the position of the sun and moon",
		[]string{},
	)

	for key, astralTime := range _astralTimes {
		setInteraction(td, key, astralTime.name, astralTime.desc)
	}

	_thing, _ = eriaServer.AddThing("", td)

	_scheduler = gocron.NewScheduler(_location)

	for key, _ := range _astralTimes {
		_thing.SetEventHandler(key, func() (interface{}, error) {
			next := _thing.GetPropertyValue("next/" + key)
			return struct{ next interface{} }{
				next: next,
			}, nil
		})
		_next[key] = setNext(key)
	}

	// Update the "/today" values each morning at 4:00
	_scheduler.Every(1).Day().At("04:00").
		Tag("main").
		StartImmediately().
		Do(updateToday)

	_scheduler.StartAsync()

	for _, job := range _scheduler.Jobs() {
		fmt.Println(job.Tags(), job.NextRun())
	}
	eriaServer.StartServer()
}

func updateToday() {
	log.Trace().Msg("[main:updateToday]")

	today := time.Now().In(_location)
	for key, astralTime := range _astralTimes {
		t := astralTime.getter(today)
		//		tStr := t.Format("15:04")
		tStr := t.Format("2006-01-02 15:04")
		_thing.SetPropertyValue("today/"+key, tStr)
	}
}

func setInteraction(td *thing.Thing, key string, name string, description string) {
	eria.AddProperty(td, "today/"+key, "-", model.PropertyDesc{
		Title:       "Today " + name + " Hour",
		Description: "Today hour when " + description,
		Type:        "string",
		Pattern:     "[0-1]{1}[0-9]{1}:[0-5]{1}[0-9]{1}",
	})

	eria.AddProperty(td, "next/"+key, "-", model.PropertyDesc{
		Title:       "Next " + name + " Time",
		Description: "Next time when " + description,
		Type:        "string",
	})

	eria.AddEvent(td, key, model.EventDesc{
		Title:       name,
		Description: description,
	})
}

func setNext(key string) *gocron.Job {
	now := time.Now().In(_location)
	tomorrow := now.Add(24 * time.Hour)

	t := _astralTimes[key].getter(now)

	if t.Before(now) {
		t = _astralTimes[key].getter(tomorrow)
	}

	// .---------------- minute (0 - 59)
	// | .-------------- hour (0 - 23)
	// | | .------------ day of month (1 - 31)
	// | | | .---------- month (1 - 12) OR jan,feb,mar ...
	// | | | | .-------- day of week (0 - 6) (Sunday=0 or 7) OR sun,mon,tue ...
	// | | | | |
	// * * * * *
	cronStr := t.Format("04 15 02 01 *")
	tStr := t.Format("2006-01-02 15:04")
	_thing.SetPropertyValue("next/"+key, tStr)
	j, _ := _scheduler.Cron(cronStr).Tag(key).Do(func(key string) {
		_thing.EmitEvent(key)
		tomorrow := now.Add(24 * time.Hour)
		t = _astralTimes[key].getter(tomorrow)
		cronStr := t.Format("04 15 02 01 *")
		_scheduler.Job(_next[key]).Cron(cronStr).Update()
		tStr := t.Format("2006-01-02 15:04")
		_thing.SetPropertyValue("next/"+key, tStr)
		for _, job := range _scheduler.Jobs() {
			fmt.Println(job.Tags(), job.NextRun())
		}
	}, key)

	return j
}