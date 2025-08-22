#ifndef MACAUDIO_H
#define MACAUDIO_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdbool.h>

// ==============================================
// Common Result Structures
// ==============================================
typedef struct {
    void* result;
    const char* error;  // NULL for success, error message for failure
} AudioEngineResult;

typedef struct {
    void* result;
    const char* error;
} AudioFormatResult;

typedef struct {
    void* result;
    const char* error;  
} AudioNodeResult;

typedef struct {
    void* result;
    const char* error;
} TapResult;

// ==============================================
// Audio Engine Structures and Functions
// ==============================================
typedef struct {
    void* engine;  // AVAudioEngine*
} AudioEngine;

// Engine lifecycle
AudioEngineResult audioengine_new(void);
void audioengine_prepare(AudioEngine* wrapper);
const char* audioengine_start(AudioEngine* wrapper);
void audioengine_stop(AudioEngine* wrapper);
void audioengine_pause(AudioEngine* wrapper);
void audioengine_reset(AudioEngine* wrapper);
const char* audioengine_is_running(AudioEngine* wrapper);
void audioengine_destroy(AudioEngine* wrapper);

// Node management
const char* audioengine_attach(AudioEngine* wrapper, void* nodePtr);
const char* audioengine_detach(AudioEngine* wrapper, void* nodePtr);

// Connection management
const char* audioengine_connect(AudioEngine* wrapper, void* sourcePtr, void* destPtr, int fromBus, int toBus);
const char* audioengine_connect_with_format(AudioEngine* wrapper, void* sourcePtr, void* destPtr, int fromBus, int toBus, void* formatPtr);
const char* audioengine_disconnect_node_input(AudioEngine* wrapper, void* nodePtr, int inputBus);
const char* audioengine_disconnect_node_output(AudioEngine* wrapper, void* nodePtr, int outputBus);

// Node access
AudioEngineResult audioengine_output_node(AudioEngine* wrapper);
AudioEngineResult audioengine_input_node(AudioEngine* wrapper);
AudioEngineResult audioengine_main_mixer_node(AudioEngine* wrapper);
AudioEngineResult audioengine_create_mixer_node(AudioEngine* wrapper);

// Volume and pan controls
const char* audioengine_set_mixer_volume(AudioEngine* wrapper, void* mixerNodePtr, float volume);
float audioengine_get_mixer_volume(AudioEngine* wrapper, void* mixerNodePtr);
void audioengine_set_mixer_pan(AudioEngine* wrapper, float pan);

// Format management
AudioEngineResult audioengine_create_format(double sampleRate, int channelCount, int bitDepth);
void audioengine_release_format(void* formatPtr);

// Engine configuration
const char* audioengine_set_buffer_size(AudioEngine* wrapper, int bufferSize);
void audioengine_remove_taps(AudioEngine* wrapper);

// ==============================================
// Audio Format Structures and Functions
// ==============================================
typedef struct {
    void* format;
} AudioFormat;

// Format creation
AudioFormatResult audioformat_new_mono(double sampleRate);
AudioFormatResult audioformat_new_stereo(double sampleRate);
AudioFormatResult audioformat_new_with_channels(double sampleRate, int channels, bool interleaved);
AudioFormatResult audioformat_new_from_spec(double sampleRate, int channels, bool interleaved);

// Format access
AudioFormatResult audioformat_get_format(AudioFormat* wrapper);
double audioformat_get_sample_rate(AudioFormat* wrapper);
int audioformat_get_channel_count(AudioFormat* wrapper);
bool audioformat_is_interleaved(AudioFormat* wrapper);

// Format operations
const char* audioformat_is_equal(AudioFormat* wrapper1, AudioFormat* wrapper2, bool* result);
void audioformat_log_info(AudioFormat* wrapper);
void audioformat_destroy(AudioFormat* wrapper);

// ==============================================
// Audio Node Functions
// ==============================================

// Generic node operations
AudioNodeResult audionode_input_format_for_bus(void* nodePtr, int bus);
AudioNodeResult audionode_output_format_for_bus(void* nodePtr, int bus);
const char* audionode_get_number_of_inputs(void* nodePtr, int* result);
const char* audionode_get_number_of_outputs(void* nodePtr, int* result);
const char* audionode_is_installed_on_engine(void* nodePtr, bool* result);
const char* audionode_log_info(void* nodePtr);
const char* audionode_release(void* nodePtr);

// Mixer node operations
AudioNodeResult audiomixer_create(void);
const char* audiomixer_set_volume(void* mixerPtr, float volume, int inputBus);
const char* audiomixer_set_pan(void* mixerPtr, float pan, int inputBus);
const char* audiomixer_get_volume(void* mixerPtr, int inputBus, float* result);
const char* audiomixer_get_pan(void* mixerPtr, int inputBus, float* result);
const char* audiomixer_release(void* mixerPtr);

// Per-connection mixer controls
const char* audiomixer_set_input_volume_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float volume);
const char* audiomixer_get_input_volume_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float* result);
const char* audiomixer_set_input_pan_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float pan);
const char* audiomixer_get_input_pan_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float* result);

// Matrix mixer operations
AudioNodeResult matrixmixer_create(void);
const char* matrixmixer_configure_invert(void* unitPtr);
const char* matrixmixer_set_gain(void* unitPtr, int inputChannel, int outputChannel, float gain);
const char* matrixmixer_get_gain(void* unitPtr, int inputChannel, int outputChannel, float* result);
const char* matrixmixer_clear_matrix(void* unitPtr);
const char* matrixmixer_set_identity(void* unitPtr);
const char* matrixmixer_set_constant_power_pan(void* unitPtr, int inputChannel, float panPosition);
const char* matrixmixer_set_linear_pan(void* unitPtr, int inputChannel, float panPosition);

// ==============================================
// Audio Tap Functions
// ==============================================

// Tap info structure
typedef struct {
    void* tapPtr;      // Unique tap identifier
    void* nodePtr;     // AVAudioNode being tapped
    int busIndex;      // Bus index being tapped
    bool isActive;     // Whether tap is currently active
    double sampleRate; // Sample rate of the tapped audio
    int channelCount;  // Number of channels being tapped
} TapInfo;

// Tap operations
void tap_init(void);
const char* tap_install(void* enginePtr, void* nodePtr, int busIndex, const char* tapKey);
const char* tap_remove(const char* tapKey);
const char* tap_get_info(const char* tapKey, TapInfo* info);
const char* tap_get_rms(const char* tapKey, double* result);
const char* tap_get_frame_count(const char* tapKey, int* result);
const char* tap_remove_all(void);
const char* tap_get_active_count(int* result);

// ==============================================
// Audio Player Functions
// ==============================================

// Player result structure
typedef struct {
    void* result;
    const char* error;  // NULL for success, error message for failure
} PlayerResult;

// Player wrapper structure  
typedef struct {
    void* playerNode;   // AVAudioPlayerNode*
    void* audioFile;    // AVAudioFile*
    void* engine;       // Reference to the engine this player belongs to
    void* timePitchUnit; // AVAudioUnitTimePitch* (nullable)
    bool isPlaying;     // Track playing state
    bool timePitchEnabled; // Whether time/pitch effects are enabled
} AudioPlayer;

// Audio buffer analysis structure
typedef struct {
    double rms_left;
    double rms_right;
    double peak_left;
    double peak_right;
    bool is_stereo;
    const char* error;  // NULL for success, error message for failure
} AudioBufferMetrics;

// Player operations
PlayerResult audioplayer_new(void* enginePtr);
const char* audioplayer_load_file(AudioPlayer* player, const char* filePath);
const char* audioplayer_play(AudioPlayer* player);
const char* audioplayer_play_at_time(AudioPlayer* player, double timeSeconds);
const char* audioplayer_pause(AudioPlayer* player);
const char* audioplayer_stop(AudioPlayer* player);
const char* audioplayer_is_playing(AudioPlayer* player, bool* result);
const char* audioplayer_get_duration(AudioPlayer* player, double* duration);
const char* audioplayer_get_current_time(AudioPlayer* player, double* currentTime);
const char* audioplayer_seek_to_time(AudioPlayer* player, double timeSeconds);
const char* audioplayer_set_volume(AudioPlayer* player, float volume);
const char* audioplayer_get_volume(AudioPlayer* player, float* volume);
const char* audioplayer_set_pan(AudioPlayer* player, float pan);
const char* audioplayer_get_pan(AudioPlayer* player, float* pan);
const char* audioplayer_set_playback_rate(AudioPlayer* player, float rate);
const char* audioplayer_get_playback_rate(AudioPlayer* player, float* rate);
const char* audioplayer_set_pitch(AudioPlayer* player, float pitch);
const char* audioplayer_get_pitch(AudioPlayer* player, float* pitch);
const char* audioplayer_enable_time_pitch_effects(AudioPlayer* player);
const char* audioplayer_disable_time_pitch_effects(AudioPlayer* player);
const char* audioplayer_is_time_pitch_effects_enabled(AudioPlayer* player, bool* enabled);
PlayerResult audioplayer_get_time_pitch_node_ptr(AudioPlayer* player);
PlayerResult audioplayer_get_node_ptr(AudioPlayer* player);
const char* audioplayer_get_file_info(AudioPlayer* player, double* sampleRate, int* channelCount, const char** format);
AudioBufferMetrics audioplayer_analyze_buffer_at_time(AudioPlayer* player, double timeSeconds);
const char* audioplayer_analyze_file_segment(AudioPlayer* player, double startTime, double duration, double* rms, int* frameCount);
void audioplayer_destroy(AudioPlayer* player);

#ifdef __cplusplus
}
#endif

#endif // MACAUDIO_H
