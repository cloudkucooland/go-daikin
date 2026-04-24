package daikin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Device struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Model           string `json:"model"`
	FirmwareVersion string `json:"firmwareVersion"`
	client          *Client
}

// actual data from device: {"mode":2,"setpointMaximum":32,"equipmentStatus":5,"tempIndoor":20.6,"equipmentCommunication":0,"humIndoor":60,"fan":0,"tempOutdoor":26,"coolSetpoint":25,"heatSetpoint":20,"modeEmHeatAvailable":false,"setpointMinimum":10,"setpointDelta":2.8,"fanCirculateSpeed":2,"fanCirculate":2,"modeLimit":1,"humOutdoor":61,"geofencingEnabled":true,"scheduleEnabled":true}

type Info struct {
	Mode                   SystemMode `json:"mode"`
	EquipmentStatus        int        `json:"equipmentStatus"`
	IndoorTemp             float64    `json:"tempIndoor"`
	EquipmentCommunication int        `json:"equipmentCommunication"`
	IndoorHumidity         int        `json:"humIndoor"`
	Fan                    FanMode    `json:"fan"`
	OutdoorTemp            float64    `json:"tempOutdoor"`
	CoolSetpoint           float64    `json:"coolSetpoint"`
	HeatSetpoint           float64    `json:"heatSetpoint"`
	ModeEMHeatAvailable    bool       `json:"modeEmHeatAvailable"`
	SetPointMinimum        float64    `json:"setpointMinimum"`
	SetPointDelta          float64    `json:"setpointDelta"`
	FanCirculateSpeed      int        `json:"fanCirculateSpeed"`
	FanCirculate           int        `json:"fanCirculate"`
	ModeLimit              int        `json:"modeLimit"`
	OutdoorHumidity        int        `json:"humOutdoor"`
	GeofencingEnabled      bool       `json:"geofencingEnabled"`
	ScheduleEnabled        bool       `json:"scheduleEnabled"`
}

type MSPPayload struct {
	Mode         SystemMode `json:"mode"`
	HeatSetpoint float64    `json:"heatSetpoint"`
	CoolSetpoint float64    `json:"coolSetpoint"`
}

func (d *Device) SetTemps(ctx context.Context, mode SystemMode, heat, cool float64) error {
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
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to set auto mode: %s: %s", res.Status, body)
	}
	return nil
}

func (d *Device) SetModeSchedule(ctx context.Context, enabled bool) error {
	url := fmt.Sprintf("/devices/%s/schedule", d.ID)
	body := struct {
		ScheduleEnabled bool `json:"scheduleEnabled"`
	}{
		ScheduleEnabled: enabled,
	}

	res, err := d.client.doRequest(ctx, "PUT", url, body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to set schedule mode: %s: %s", res.Status, body)
	}
	return nil
}

func (d *Device) GetInfo(ctx context.Context) (*Info, error) {
	url := fmt.Sprintf("/devices/%s", d.ID)
	res, err := d.client.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to get device info: %s: %s", res.Status, body)
	}

	var info Info
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode info: %w", err)
	}
	return &info, nil
}

// FAN CONTROL
type FanPayload struct {
	Circulate int `json:"fanCirculate"`      // 0: Off, 1: On, 2: schedule
	Speed     int `json:"fanCirculateSpeed"` // 0 low, 1 medium, 2 high
}

func (d *Device) SetFan(ctx context.Context, circulate int, speed int) error {
	url := fmt.Sprintf("/devices/%s/fan", d.ID)
	payload := FanPayload{
		Circulate: circulate,
		Speed:     speed,
	}

	res, err := d.client.doRequest(ctx, "PUT", url, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to set fan mode: %s: %s", res.Status, body)
	}
	return nil
}
