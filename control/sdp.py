import dbus
import dbus.mainloop.glib
import os
import sys
import time

# Read service discovery protocol from standard input.
service_record = sys.stdin.read()

# Register profile. Per https://goo.gl/7TvNGG and https://goo.gl/s9sauE, this
# UUID corresponds to an HID device.
bus = dbus.SystemBus()
manager = dbus.Interface(bus.get_object("org.bluez","/org/bluez"), "org.bluez.ProfileManager1")
uuid="00001124-0000-1000-8000-00805f9b34fb"
opts = {
	"ServiceRecord": service_record,
	"Role": "server",
	"RequireAuthentication": False,
	"RequireAuthorization": False,
}
manager.RegisterProfile("/org/bluez/ergoblue", uuid, opts)

dbus.mainloop.glib.DBusGMainLoop(set_as_default=True)
time.sleep(sys.maxsize)
