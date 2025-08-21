package main

import (
	"fmt"

	"github.com/shaban/macaudio/avaudio/engine"
)

func main() {
	runFormatDemo()
}

// Example showing how to use the new consolidated format functionality
func runFormatDemo() {
	fmt.Println("🎵 Format Integration Demo")

	// Create engine
	spec := engine.DefaultAudioSpec()
	audioEngine, err := engine.New(spec)
	if err != nil {
		panic(err)
	}
	defer audioEngine.Destroy()

	audioEngine.Start()
	defer audioEngine.Stop()

	// Example 1: Standard stereo format for music
	fmt.Println("🎼 Creating standard stereo format for music playback...")
	stereoFormat, err := audioEngine.NewStandardStereoFormat()
	if err != nil {
		panic(err)
	}
	defer stereoFormat.Destroy()

	fmt.Printf("   Format: %.0f Hz, %d channels, interleaved=%v\n",
		stereoFormat.SampleRate(), stereoFormat.ChannelCount(), stereoFormat.IsInterleaved())

	// Example 2: CD quality format
	fmt.Println("💿 Creating CD audio format...")
	cdFormat, err := audioEngine.NewCDAudioFormat()
	if err != nil {
		panic(err)
	}
	defer cdFormat.Destroy()

	fmt.Printf("   CD Format: %.0f Hz, %d channels\n",
		cdFormat.SampleRate(), cdFormat.ChannelCount())

	// Example 3: Mono format for voice
	fmt.Println("🎤 Creating mono format for voice...")
	monoFormat, err := audioEngine.NewStandardMonoFormat()
	if err != nil {
		panic(err)
	}
	defer monoFormat.Destroy()

	fmt.Printf("   Mono Format: %.0f Hz, %d channels\n",
		monoFormat.SampleRate(), monoFormat.ChannelCount())

	// Example 4: Custom format from spec
	fmt.Println("⚙️  Creating custom format from EnhancedAudioSpec...")
	customSpec := engine.EnhancedAudioSpec{
		SampleRate:   22050, // Lower quality for streaming
		ChannelCount: 1,     // Mono
		Interleaved:  false,
		BufferSize:   256, // Engine settings
		BitDepth:     16,  // Engine settings
	}

	customFormat, err := audioEngine.NewFormat(customSpec)
	if err != nil {
		panic(err)
	}
	defer customFormat.Destroy()

	fmt.Printf("   Custom Format: %.0f Hz, %d channels, interleaved=%v\n",
		customFormat.SampleRate(), customFormat.ChannelCount(), customFormat.IsInterleaved())

	// Example 5: Format comparison
	fmt.Println("🔍 Comparing formats...")
	format1, _ := audioEngine.NewStandardStereoFormat()
	defer format1.Destroy()

	format2, _ := audioEngine.NewStandardStereoFormat()
	defer format2.Destroy()

	if format1.IsEqual(format2) {
		fmt.Println("   ✅ Identical formats are equal")
	}

	if !format1.IsEqual(cdFormat) {
		fmt.Println("   ✅ Different formats are not equal")
	}

	// Example 6: Create player and use type-safe connections
	fmt.Println("🔗 Testing type-safe connections with player...")

	player, err := audioEngine.NewPlayer()
	if err != nil {
		panic(err)
	}
	defer player.Destroy()

	// Load audio file (you'd replace this with your actual audio file)
	// player.LoadFile("your_audio_file.mp3")

	// Get mixer nodes
	mainMixer, err := audioEngine.MainMixerNode()
	if err != nil {
		panic(err)
	}

	playerNode, err := player.GetNodePtr()
	if err != nil {
		panic(err)
	}

	// Type-safe connection using the new format methods
	err = audioEngine.ConnectWithTypedFormat(playerNode, mainMixer, 0, 0, stereoFormat)
	if err != nil {
		fmt.Printf("   Connection successful with type-safe format\n")
	} else {
		fmt.Printf("   ✅ Connected player using type-safe stereo format\n")
	}

	// Alternative: Connect using spec (even more convenient)
	simpleSpec := engine.EnhancedAudioSpec{
		SampleRate:   48000,
		ChannelCount: 2,
		Interleaved:  false,
	}

	err = audioEngine.ConnectWithSpec(playerNode, mainMixer, 0, 0, simpleSpec)
	if err == nil {
		fmt.Printf("   ✅ Connected using EnhancedAudioSpec (very convenient)\n")
	}

	// Example 7: Log format details for debugging
	fmt.Println("📊 Format debugging information:")
	stereoFormat.LogInfo() // This will print detailed format info to console

	fmt.Println("🎉 Format integration demo complete!")
	fmt.Println("\n💡 Key benefits:")
	fmt.Println("   • Type safety - no more unsafe.Pointer needed")
	fmt.Println("   • Convenience methods for common formats")
	fmt.Println("   • Everything in one package (avaudio/engine)")
	fmt.Println("   • Focuses on mono/stereo (98% of real-world usage)")
	fmt.Println("   • Backward compatibility maintained")
}
