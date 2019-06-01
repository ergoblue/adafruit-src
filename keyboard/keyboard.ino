#include <bluefruit.h>

// Define bitmap for rows with HIGH voltage.
volatile uint64_t rising = 0;

// When an event occurs, set the corresponding least significant bit. Resume
// loop if currently suspended. Call resumeLoop() regardless of whether loop()
// is running; it should have no effect if it already is. TODO consider a
// variable to store the status of loop(). GPIOTE_IRQHandler() appears to run
// the interrupt functions synchronously.
void on_rising(uint8_t pin) {
	rising |= 1 << pin;
	resumeLoop();
}

// Define interrupt function for each row.
#define ON_RISING(X) \
void on_rising_##X() { \
	on_rising(X); \
}
ON_RISING(0); ON_RISING(1); ON_RISING(2); ON_RISING(3); ON_RISING(4); ON_RISING(5);

// Define rows and columns.
struct row_t {
	uint8_t pin;
	void (*event)();
};
struct row_t rows[] = {
	{A1, on_rising_0}, {A2, on_rising_1}, {A3, on_rising_2},
	{A4, on_rising_3}, {A5, on_rising_4}, {A0, on_rising_5},
};
int row_count = sizeof(rows) / sizeof(rows[0]);
#ifdef NRF52840_XXAA
uint8_t col_pins[] = {12, 11, 10, 9, 6, 5, PIN_WIRE_SCL};
#else
uint8_t col_pins[] = {15, 7, 11, 31, 30, 27, PIN_WIRE_SCL};
#endif
int col_count = sizeof(col_pins) / sizeof(col_pins[0]);

// Define Bluetooth service for keyboard.
BLEHidGeneric blehid = BLEHidGeneric(1, 0, 0);

// Define MAC address of the central controllers. Note that the bytes are in the
// reverse order of the printable address.
uint8_t bos_mac[] = {0x00, 0x00, 0x00, 0x00, 0x00, 0x00};
uint8_t bwi_mac[] = {0x01, 0x01, 0x01, 0x01, 0x01, 0x01};

// Keep track of Bluetooth connection status.
bool advertising = false;
bool connected = false;

void advertise() {
	advertising = true;
	Bluefruit.Advertising.start(5);
}

void connect_callback(uint16_t hdl) {
	// If MAC address matches, update connection status. Otherwise, disconnect
	// and advertise again. Ideally we would not accept unrecognized connections
	// in the first place but we would have to customize Bluefruit to do that.
	ble_gap_addr_t raddr = Bluefruit.getPeerAddr();
	if (memcmp(raddr.addr, bos_mac, 6) == 0 || memcmp(raddr.addr, bwi_mac, 6) == 0) {
		advertising = false;
		connected = true;
	} else {
		Bluefruit.disconnect();
		advertise();
	}
}

void disconnect_callback(uint16_t hdl, uint8_t reason) {
	advertising = true;
	connected = false;
	advertise();
}

void stop_callback() {
	advertising = false;
}

void setup() {
	// Set rows to be input with interrupt for each.
	for (int i = 0; i < row_count; i++) {
		pinMode(rows[i].pin, INPUT_PULLDOWN);
		attachInterrupt(digitalPinToInterrupt(rows[i].pin), rows[i].event, RISING);
	}

	// Set columns to be output and initialize each to high.
	for (int i = 0; i < col_count; i++) {
		pinMode(col_pins[i], OUTPUT);
		digitalWrite(col_pins[i], HIGH);
	}

	// Set Bluetooth slave latency to reduce power consumption.
	Bluefruit.setSlaveLatency(24);

	// Initialize Bluetooth.
	Bluefruit.begin();

	// Enable DC DC mode. This appears to reduce power consumption by a good
	// amount. It only makes a difference after enabling Bluefruit.
	sd_power_dcdc_mode_set(NRF_POWER_DCDC_ENABLE);
	
	// Include the MAC address in device name so we can distinguish the left
	// hand from the right hand on the controller using HIDIOCGRAWNAME. Ideally
	// we would use HIDIOCGRAWPHYS to get the MAC address directly, though that
	// seems to give the controller MAC address, as reproduced in
	// https://goo.gl/ptkRmd.
	uint8_t mac[6];
	Bluefruit.Gap.getAddr(mac);
	char name[21];
	char * fmt = "ErgoBlue %02x%02x%02x%02x%02x%02x";
	sprintf(name, fmt, mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
	Bluefruit.setName(name);

	// Do not display led when Bluetooth connection is active.
	Bluefruit.autoConnLed(false);

	// Define HID report. We are merely sending our own data over the HID
	// protocol and make no effort to emulate an actual keyboard.
	uint8_t const hid_report[] = {
		HID_USAGE_PAGE(HID_USAGE_PAGE_DESKTOP),
		HID_USAGE(HID_USAGE_DESKTOP_KEYBOARD),
		HID_COLLECTION(HID_COLLECTION_APPLICATION),
			HID_REPORT_ID(1),
			HID_REPORT_COUNT(7),
			HID_REPORT_SIZE(8),
			HID_INPUT(HID_CONSTANT),
		HID_COLLECTION_END,
	};

	// Initialize keyboard based on code in BLEHidAdafruit.cpp.
	uint16_t input_len [] = {7};
	blehid.setReportLen(input_len, NULL, NULL);
	blehid.enableKeyboard(true);
	blehid.setReportMap(hid_report, sizeof(hid_report));
	blehid.begin();

	// Use 10ms as the connection interval. The interval is defined as some
	// multiple of 1.25ms.
	Bluefruit.setConnInterval(8, 8);

	// Use Bluetooth Low Energy without BR/EDR.
	Bluefruit.Advertising.addFlags(BLE_GAP_ADV_FLAGS_LE_ONLY_GENERAL_DISC_MODE);

	// Set device to be keyboard.
	Bluefruit.Advertising.addAppearance(BLE_APPEARANCE_HID_KEYBOARD);

	// Include BLE HID service.
	Bluefruit.Advertising.addService(blehid);

	// Include device name in the advertising packet.
	Bluefruit.Advertising.addName();
	
	// Do not automatically advertise when the central disconnects.
	Bluefruit.Advertising.restartOnDisconnect(false);

	// Advertise every 100ms for 1 second and subsequently every 400ms.
	Bluefruit.Advertising.setInterval(160, 640);
	Bluefruit.Advertising.setFastTimeout(1);

	// Setup callback functions to keep track of Bluetooth state.
	Bluefruit.setConnectCallback(connect_callback);
	Bluefruit.setDisconnectCallback(disconnect_callback);
	Bluefruit.Advertising.setStopCallback(stop_callback);

	// Begin advertising.
	advertise();

	// Suspend loop until event.
	suspendLoop();
}

void loopInternal() {
	// Set all columns to LOW.
	for (int i = 0; i < col_count; i++) {
		digitalWrite(col_pins[i], LOW);
	}

	// Define array for keyboard bitmap. By keeping the result of the last few
	// scans, we can minimize the latency of key down events.
	const int bitmap_len = 4;
	uint64_t bitmap[bitmap_len] = {};
	int i = 0;

	while (true) {
		// Toggle voltage for each column. Add each result to bitmap. It seems
		// that we need a 1ms delay when writing to the A7 pin, which is also
		// used to detect the battery voltage.
		for (int j = 0; j < col_count; j++) {
			rising = 0;
			digitalWrite(col_pins[j], HIGH);
			if (col_pins[j] == PIN_A7) {
				delay(1);
			}
			digitalWrite(col_pins[j], LOW);
			bitmap[i] |= rising << (j * row_count);
		}

		// OR everything except the last for bimap and everything except the
		// first for bitmap1. This lets us detect key down events as soon as
		// they occur and deal with debouncing later. This does introduce a
		// small latency for key up events.
		uint64_t bitmap0 = bitmap[(i+1)%bitmap_len];
		uint64_t bitmap1 = bitmap[i];
		for (int j = 2; j < bitmap_len; j++) {
			uint64_t bitmap2 = bitmap[(i+j)%bitmap_len];
			bitmap0 |= bitmap2;
			bitmap1 |= bitmap2;
		}

		// If the keys changed, send new bitmap to control server.
		if (bitmap0 != bitmap1) {
			// Convert bits to unsigned character array.
			unsigned char data[sizeof(bitmap1)];
			memcpy(data, &bitmap1, sizeof(bitmap1));

			// Send data if connected to central. Otherwise, advertise if not
			// already advertising.
			if (connected) {
				blehid.inputReport(1, data, 7);
			} else if (!advertising) {
				advertise();
			}
		}

		// If no key is active, stop matrix scanning.
		if (bitmap1 == 0) {
			break;
		}

		// Increment the bitmap array position.
		i++;
		i %= bitmap_len;
		bitmap[i] = 0;

		// Sleep for 5ms until next scan.
		delay(5);
	}
}

void loop() {
	while (true) {
		loopInternal();

		// Set all columns to HIGH to get event when a new key is active. It is
		// possible that a key was pressed between when its column was last
		// scanned and here. Give it a second so we can detect those keys. If
		// any such key exists, continue processing.
		for (int i = 0; i < col_count; i++) {
			digitalWrite(col_pins[i], HIGH);
		}
		delay(1);
		if (rising == 0) {
			break;
		}
	}

	// Suspend loop until next event.
	suspendLoop();
}
