package main

import (
	"time"

	"github.com/project-eria/go-wot/dataSchema"
	"github.com/project-eria/go-wot/interaction"
	"github.com/project-eria/go-wot/producer"
	"github.com/project-eria/go-wot/thing"
	"github.com/sj14/astral/pkg/astral"

	"github.com/go-co-op/gocron/v2"

	eria "github.com/project-eria/eria-core"
	zlog "github.com/rs/zerolog/log"
)

type AstralInfo struct {
	name   string
	desc   string
	getter func(time.Time) time.Time
}

var (
	// Version is a placeholder that will receive the git tag version during build time
	Version      = "-"
	_thing       producer.ExposedThing
	_observer    astral.Observer
	_astralTimes map[string]AstralInfo
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
	defer func() {
		zlog.Info().Msg("[main] Stopped")
	}()

	eria.Init("ERIA Ephemeris Info", &config)
	_observer = astral.Observer{Latitude: config.Lat, Longitude: config.Long, Elevation: 0}
	_astralTimes = map[string]AstralInfo{
		"dawnAstronomical": {
			name: "Dawn (Astronomical)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dawn(_observer, t, astral.DepressionAstronomical)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:dawnAstronomical]")
				}
				return value
			},
		},
		"dawnNautical": {
			name: "Dawn (Nautical)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dawn(_observer, t, astral.DepressionNautical)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:dawnNautical]")
				}
				return value
			},
		},
		"dawnCivil": {
			name: "Dawn (Civil)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dawn(_observer, t, astral.DepressionCivil)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:dawnCivil]")
				}
				return value
			},
		},
		"goldenHourRisingStart": {
			name: "Golden Hour Start (Rising)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, _, err := astral.GoldenHour(_observer, t, astral.SunDirectionRising)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:goldenHourRisingStart]")
				}
				return value
			},
		},
		"sunrise": {
			name: "Sunrise",
			desc: "the Sun appears on the horizon in the morning",
			getter: func(t time.Time) time.Time {
				value, err := astral.Sunrise(_observer, t)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:sunrise]")
				}
				return value
			},
		},
		"goldenHourRisingEnd": {
			name: "Golden Hour End (Rising)",
			desc: "",
			getter: func(t time.Time) time.Time {
				_, value, err := astral.GoldenHour(_observer, t, astral.SunDirectionRising)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:goldenHourRisingEnd]")
				}
				return value
			},
		},
		"noon": {
			name: "noon",
			desc: "",
			getter: func(t time.Time) time.Time {
				value := astral.Noon(_observer, t)
				return value
			},
		},
		"goldenHourSettingStart": {
			name: "Golden Hour Start (Setting)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, _, err := astral.GoldenHour(_observer, t, astral.SunDirectionSetting)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:goldenHourSettingStart]")
				}
				return value
			},
		},
		"sunset": {
			name: "Sunset",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Sunset(_observer, t)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:sunset]")
				}
				return value
			},
		},
		"goldenHourSettingEnd": {
			name: "Golden Hour End (Setting)",
			desc: "",
			getter: func(t time.Time) time.Time {
				_, value, err := astral.GoldenHour(_observer, t, astral.SunDirectionSetting)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:goldenHourSettingEnd]")
				}
				return value
			},
		},
		"duskCivil": {
			name: "Dusk (Civil)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dusk(_observer, t, astral.DepressionCivil)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:duskCivil]")
				}
				return value
			},
		},
		"duskNautical": {
			name: "Dusk (Nautical)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dusk(_observer, t, astral.DepressionNautical)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:duskNautical]")
				}
				return value
			},
		},
		"duskAstronomical": {
			name: "Dusk (Astronomical)",
			desc: "",
			getter: func(t time.Time) time.Time {
				value, err := astral.Dusk(_observer, t, astral.DepressionAstronomical)
				if err != nil {
					zlog.Error().Err(err).Msg("[init:duskAstronomical]")
				}
				return value
			},
		},
		"midnight": {
			name: "Midnight",
			desc: "",
			getter: func(t time.Time) time.Time {
				value := astral.Midnight(_observer, t)
				return value
			},
		},
	}
}

func main() {
	defer func() {
		zlog.Info().Msg("[main] Stopped")
	}()

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

	eriaProducer := eria.Producer("")
	_thing, _ = eriaProducer.AddThing("", td)

	for key := range _astralTimes {
		eriaProducer.PropertyUseDefaultHandlers(_thing, "today/"+key)
		eriaProducer.PropertyUseDefaultHandlers(_thing, "next/"+key)
		_thing.SetEventHandler(key, func() (interface{}, error) {
			next := eriaProducer.GetPropertyValue(_thing, "next/"+key)
			return struct{ next interface{} }{
				next: next,
			}, nil
		})
		initNext(key)
	}

	scheduler := eria.GetCronScheduler()

	// Update the "/today" values each morning at 0:00
	scheduler.NewJob(
		gocron.DailyJob(
			1,
			gocron.NewAtTimes(
				gocron.NewAtTime(0, 0, 0),
			),
		),
		gocron.NewTask(updateToday),
		gocron.WithTags("refresh", "main"),
		gocron.WithStartAt(
			gocron.WithStartImmediately(),
		),
	)

	eria.Start("")
}

func updateToday() {
	today := time.Now().In(eria.Location())
	eriaProducer := eria.Producer("")
	for key, astralTime := range _astralTimes {
		t := astralTime.getter(today)
		tStr := t.Format(time.RFC3339)
		eriaProducer.SetPropertyValue(_thing, "today/"+key, tStr)
	}
}

func setInteraction(td *thing.Thing, key string, name string, description string) {
	dateString, _ := dataSchema.NewString(
		dataSchema.StringPattern("[0-1]{1}[0-9]{1}:[0-5]{1}[0-9]{1}"),
	)
	td.AddProperty(interaction.NewProperty(
		"today/"+key,
		"Today "+name+" Hour",
		"Today hour when "+description,
		dateString,
		interaction.PropertyReadOnly(true),
	))

	td.AddProperty(interaction.NewProperty(
		"next/"+key,
		"Next "+name+" Time",
		"Next time when "+description,
		dateString,
		interaction.PropertyReadOnly(true),
	))

	td.AddEvent(interaction.NewEvent(
		key,
		name,
		description,
		interaction.EventData(&dateString),
	))
}

func initNext(key string) {
	now := time.Now().In(eria.Location())
	t := _astralTimes[key].getter(now)
	if t.Before(now) {
		tomorrow := now.Add(24 * time.Hour)
		t = _astralTimes[key].getter(tomorrow)
	}

	setNext(key, t)
}

func runNext(key string) {
	zlog.Trace().Str("time", key).Msg("[main:runNext]")
	_thing.EmitEvent(key, nil)
	now := time.Now().In(eria.Location())
	tomorrow := now.Add(24 * time.Hour)
	t := _astralTimes[key].getter(tomorrow)

	setNext(key, t)
}

func setNext(key string, t time.Time) {
	tStr := t.Format(time.RFC3339)
	eriaProducer := eria.Producer("")
	eriaProducer.SetPropertyValue(_thing, "next/"+key, tStr)
	scheduler := eria.GetCronScheduler()

	scheduler.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(t),
		),
		gocron.NewTask(runNext, key),
		gocron.WithTags(key, "main"),
	)
}
