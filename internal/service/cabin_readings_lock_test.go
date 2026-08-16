package service

import (
	"testing"
	"time"

	"batteryops/internal/domain"
)

// TestReadingsDuringMaintenanceKeepCabinLock describes the expected public
// behaviour when new cabin measurements keep arriving while a maintenance work
// order for that cabin is already in flight.
//
// Input: a cabin is registered, an alarm is reported and the maintenance order
// is dispatched (which locks the cabin).  Afterwards the cabin keeps pushing
// over-limit measurements, first while the order is dispatched and again while
// the engineer is on site.
//
// Expected output: every measurement is stored and readable, the cabin stays
// bound to the dispatched order for the whole repair, the engineer can arrive
// and finish the repair, and the cabin returns to normal only after the dual
// acceptance closes the order.
func TestReadingsDuringMaintenanceKeepCabinLock(t *testing.T) {
	svc, st, _ := newTestService(30 * time.Minute)
	registerCabin(svc, "cabin-lock-1")

	result, err := svc.ReportAlarm("cabin-lock-1", domain.AlarmTempOverLimit, "inspector-1")
	if err != nil {
		t.Fatalf("report alarm: %v", err)
	}
	orderID := result.Order.ID

	if _, err := svc.DispatchMaintenance(orderID, "dispatcher-1", "eng-1", "eng-2"); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	cabin, _ := st.GetCabin("cabin-lock-1")
	if cabin.Status != domain.CabinLocked || cabin.LockOrderID != orderID {
		t.Fatalf("after dispatch: expected status=%s lock_order_id=%s, got status=%s lock_order_id=%q",
			domain.CabinLocked, orderID, cabin.Status, cabin.LockOrderID)
	}

	// The cabin keeps reporting an over-limit temperature while the engineer is
	// still on the way.
	cabin, err = svc.SubmitReadings("cabin-lock-1", domain.Readings{
		Temperature:  61.5,
		Humidity:     44.0,
		Voltage:      51.5,
		InsulationOK: true,
	})
	if err != nil {
		t.Fatalf("submit readings while dispatched: %v", err)
	}
	if cabin.Readings == nil || cabin.Readings.Temperature != 61.5 {
		t.Fatalf("expected the 61.5 reading to be stored, got %+v", cabin.Readings)
	}
	if cabin.Status != domain.CabinLocked {
		t.Fatalf("readings arriving during a dispatched repair must not change the cabin status: want %s, got %s",
			domain.CabinLocked, cabin.Status)
	}
	if cabin.LockOrderID != orderID {
		t.Fatalf("expected cabin to stay bound to order %s, got %q", orderID, cabin.LockOrderID)
	}

	// The assigned engineer must still be able to start the repair.
	order, err := svc.EngineerArrive(orderID, "eng-1")
	if err != nil {
		t.Fatalf("engineer arrival after new readings: %v", err)
	}
	if order.Status != domain.MaintProcessing {
		t.Fatalf("expected order status %s, got %s", domain.MaintProcessing, order.Status)
	}
	cabin, _ = st.GetCabin("cabin-lock-1")
	if cabin.Status != domain.CabinUnderMaintenance {
		t.Fatalf("expected cabin status %s, got %s", domain.CabinUnderMaintenance, cabin.Status)
	}

	// Another over-limit measurement while the engineer works on site.
	cabin, err = svc.SubmitReadings("cabin-lock-1", domain.Readings{
		Temperature:  58.0,
		Humidity:     43.0,
		Voltage:      51.0,
		InsulationOK: false,
	})
	if err != nil {
		t.Fatalf("submit readings while under maintenance: %v", err)
	}
	if cabin.Status != domain.CabinUnderMaintenance {
		t.Fatalf("readings arriving during on-site repair must not change the cabin status: want %s, got %s",
			domain.CabinUnderMaintenance, cabin.Status)
	}
	if cabin.Readings == nil || cabin.Readings.Temperature != 58.0 || cabin.Readings.InsulationOK {
		t.Fatalf("expected the 58.0 insulation-failed reading to be stored, got %+v", cabin.Readings)
	}

	if _, err := svc.CompleteProcessing(orderID, "eng-1"); err != nil {
		t.Fatalf("complete processing: %v", err)
	}
	if _, err := svc.AcceptMaintenance(orderID, "leader-1", "station-1"); err != nil {
		t.Fatalf("dual acceptance: %v", err)
	}

	cabin, _ = st.GetCabin("cabin-lock-1")
	if cabin.Status != domain.CabinNormal {
		t.Fatalf("expected cabin status %s after closure, got %s", domain.CabinNormal, cabin.Status)
	}
	if cabin.LockOrderID != "" {
		t.Fatalf("expected lock_order_id to be cleared after closure, got %q", cabin.LockOrderID)
	}
}

// TestReadingsOnNormalCabinStillRaiseAlarm keeps the ordinary alarm path
// covered: an idle cabin that reports an over-limit temperature or an
// insulation failure must move into the alarm state.
func TestReadingsOnNormalCabinStillRaiseAlarm(t *testing.T) {
	svc, _, _ := newTestService(30 * time.Minute)
	registerCabin(svc, "cabin-lock-2")

	cabin, err := svc.SubmitReadings("cabin-lock-2", domain.Readings{
		Temperature:  49.0,
		Humidity:     40.0,
		Voltage:      52.0,
		InsulationOK: true,
	})
	if err != nil {
		t.Fatalf("submit readings: %v", err)
	}
	if cabin.Status != domain.CabinAlarm {
		t.Fatalf("expected status %s, got %s", domain.CabinAlarm, cabin.Status)
	}

	registerCabin(svc, "cabin-lock-3")
	cabin, err = svc.SubmitReadings("cabin-lock-3", domain.Readings{
		Temperature:  30.0,
		Humidity:     40.0,
		Voltage:      52.0,
		InsulationOK: false,
	})
	if err != nil {
		t.Fatalf("submit readings: %v", err)
	}
	if cabin.Status != domain.CabinAlarm {
		t.Fatalf("expected status %s for insulation failure, got %s", domain.CabinAlarm, cabin.Status)
	}
}
