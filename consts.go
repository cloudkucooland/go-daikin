package daikin

import (
	"time"
)

const base = "https://integrator-api.daikinskyport.com/v1"
const httpTimeout = 10 * time.Second

type SystemMode int

const ModeOff SystemMode = 0
const ModeHeat SystemMode = 1
const ModeCool SystemMode = 2
const ModeAuto SystemMode = 3

type FanMode int

const FanModeOff FanMode = 0
const FanModeOn FanMode = 1
const FanModeAuto FanMode = 2

type AwayMode int

const AwayModeHome AwayMode = 0
const AwayModeAway AwayMode = 1
