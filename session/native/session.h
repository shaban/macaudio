#ifndef MACAUDIO_SESSION_H
#define MACAUDIO_SESSION_H

#ifdef __cplusplus
extern "C" {
#endif

// Go callback function
extern void configurationChanged(void);

// C functions called from Go
void macaudio_setup_config_monitoring(void* enginePtr);
void macaudio_cleanup_config_monitoring(void);

// Test function to simulate hotplug events
void macaudio_simulate_hotplug(void* enginePtr);

#ifdef __cplusplus
}
#endif

#endif // MACAUDIO_SESSION_H