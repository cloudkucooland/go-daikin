package daikin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type Device struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Model           string `json:"model"`
	FirmwareVersion string `json:"firmwareVersion"`
	client          *Client
}

type MSPPayload struct {
	Mode         Mode    `json:"mode"`
	HeatSetpoint float64 `json:"heatSetpoint"`
	CoolSetpoint float64 `json:"coolSetpoint"`
}

type Info struct {
	EquipmentStatus     int     `json:"equipmentStatus"`
	Mode                Mode    `json:"mode"`
	ModeLimit           int     `json:"modeLimit"`
	ModeEMHeatAvailable bool    `json:"modeEmHeatAvailable"`
	Fan                 int     `json:"fan"`
	FanCirculate        int     `json:"fanCirculate"`
	FanCirculateSpeed   int     `json:"fanCirculateSpeed"`
	HeatSetpoint        float64 `json:"heatSetpoint"`
	CoolSetpoint        float64 `json:"coolSetpoint"`
	SetPointDelta       int     `json:"setpointDelta"`
	SetPointMinimum     float64 `json:"setpointMinimum"`
	SetPointMaximum     float64 `json:"setpointMaximum"`
	IndoorTemp          float64 `json:"tempIndoor"`
	IndoorHumidity      int     `json:"humIndoor"`
	OutdoorTemp         float64 `json:"tempOutdoor"`
	OutdoorHumidity     int     `json:"humOutdoor"`
	ScheduleEnabled     bool    `json:"scheduleEnabled"`
	GeofencingEnabled   bool    `json:"geofencingEnabled"`
	CoolNextPeriod      int     `json:"coolnextperiod"`
	ActiveError         string  `json:"activeerror"`
	Humidification      string  `json:"humidification"`
	DehumSetpoint       int     `json:"dehumSetpoint"`
	DRIsActive          bool    `json:"drIsActive"`
	DROffsetDegree      float64 `json:"drOffsetDegree"`
}

func (d *Device) SetTemps(ctx context.Context, mode Mode, heat, cool float64) error {
	url := fmt.Sprintf("/devices/%s/msp", d.ID)

	body := MSPPayload{
		Mode:         mode,
		HeatSetpoint: heat,
		CoolSetpoint: cool,
	}

	res, err := d.client.doRequest(ctx, "PUT", url, body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set auto mode: %s", res.Status)
	}

	slog.Info("Daikin set to deep cool", "device", d.ID, "mode", mode, "cool", cool, "heat", heat)
	return nil
}

func (d *Device) SetModeSchedule(ctx context.Context) error {
	url := fmt.Sprintf("/devices/%s/schedule", d.ID)
	body := struct {
		ScheduleEnabled bool `json:"scheduleEnabled"`
	}{
		ScheduleEnabled: true,
	}

	res, err := d.client.doRequest(ctx, "PUT", url, body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to toggle schedule: %s", res.Status)
	}

	// slog.Info("Daikin set to Schedule", "device", d.ID)
	return nil
}

func (d *Device) GetInfo(ctx context.Context) (*Info, error) {
	url := fmt.Sprintf("/devices/%s", d.ID)
	res, err := d.client.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var info Info
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode info: %w", err)
	}

	// slog.Info("Daikin data pulled", "indoor temp", info.IndoorTemp, "indoor humidity", info.IndoorHumidity)
	return &info, nil
}

// FAN CONTROL
type FanPayload struct {
	Mode      int `json:"fanMode"`      // 0: Auto, 1: On, 2: Circulate
	Circulate int `json:"fanCirculate"` // 0: Off, 1: On
}

func (d *Device) SetFan(ctx context.Context, mode int, circulate int) error {
	url := fmt.Sprintf("/devices/%s/fan", d.ID)
	payload := FanPayload{
		Mode:      mode,
		Circulate: circulate,
	}

	res, err := d.client.doRequest(ctx, "PUT", url, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to set fan: %s", res.Status)
	}
	return nil
}

// AWAY MODE
type AwayPayload struct {
	AwayMode int `json:"awayMode"` // 0: Home, 1: Away
}

func (d *Device) SetAwayMode(ctx context.Context, away int) error {
	url := fmt.Sprintf("/devices/%s/away", d.ID)
	payload := AwayPayload{
		AwayMode: away,
	}

	res, err := d.client.doRequest(ctx, "PUT", url, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to set away mode: %s", res.Status)
	}
	return nil
}

// DEMAND RESPONSE
type DRPayload struct {
	IsActive     bool    `json:"isActive"` // is this really bool? all the others are int 0/1
	OffsetDegree float64 `json:"offsetDegree"`
}

func (d *Device) SetDemandResponse(ctx context.Context, active bool, offset float64) error {
	url := fmt.Sprintf("/devices/%s/dr", d.ID)
	payload := DRPayload{
		IsActive:     active,
		OffsetDegree: offset,
	}

	res, err := d.client.doRequest(ctx, "PUT", url, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to set demand response: %s", res.Status)
	}
	return nil
}

type DehumPayload struct {
	DehumSetpoint int `json:"dehumSetpoint"`
}

func (d *Device) SetDehumidifySetpoint(ctx context.Context, setpoint int) error {
	// Validation
	if setpoint < 35 || setpoint > 80 {
		return fmt.Errorf("dehum setpoint %d is out of range (35-80)", setpoint)
	}

	url := fmt.Sprintf("/devices/%s/dehum", d.ID)
	payload := DehumPayload{
		DehumSetpoint: setpoint,
	}

	res, err := d.client.doRequest(ctx, "PUT", url, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to set dehumidify setpoint: %s", res.Status)
	}
	return nil
}
