# AVAudio Package Migration Plan
## String-Based Error Handling & Native File Separation

Based on the successful migration of `avaudio/engine` and `avaudio/format`, this document outlines the migration plan for all other `avaudio` subpackages to use the same architecture:

- **C/ObjC code in `.m` files** (implementations + struct definitions)
- **Function declarations in CGO block** (no separate .h file needed!)  
- **String-based error handling** (NULL = success, string = error message)
- **Clean Go error mapping**

---

## 🔑 **Key Discovery: CGO Pattern**

**IMPORTANT**: We discovered the correct CGO pattern from working packages like `devices` and `format`:

### **Correct File Structure:**
```
avaudio/[package]/
├── [package].go           # Pure Go code + CGO block with function declarations
├── [package]_test.go      # Go tests  
└── native/
    └── [package].m        # C/ObjC implementations + struct definitions
```

**No `.h` file needed!** CGO can resolve struct types directly from the `.m` file.

### **Correct CGO Block Pattern:**
```go
/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include "native/[package].m"
#include <stdlib.h>

// Function declarations ONLY - CGO resolves structs from .m file
AudioResult audiofunction_create(double param);
const char* audiofunction_process(void* ptr);
void audiofunction_destroy(void* ptr);
*/
import "C"
```

**Key insights:**
- ✅ **Include the `.m` file directly**: `#include "native/[package].m"`  
- ✅ **Declare functions in CGO block**: Tells CGO what functions to expose
- ✅ **Let CGO resolve structs**: No need to duplicate struct definitions
- ✅ **Single source of truth**: `.m` file contains both structs and implementations

---

## 🎯 **Migration Pattern Template**

### **File Structure After Migration:**
```
avaudio/[package]/
├── [package].go           # Pure Go code + CGO function declarations
├── [package]_test.go      # Go tests  
└── native/
    └── [package].m        # C/ObjC implementations + struct definitions
```

### **CGO Block Template:**
```go
/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include "native/[package].m"
#include <stdlib.h>

// Function declarations - CGO resolves structs from .m file
AudioPackageResult audiopackage_create(double param);
const char* audiopackage_process(void* ptr);
void audiopackage_destroy(void* ptr);
*/
import "C"
```

### **Error Handling Patterns:**

#### **Pattern A: Functions Returning Pointers**
```c
// In .m file - struct definitions + implementations
typedef struct {
    void* result;           // The actual result pointer
    const char* error;      // NULL = success, string = error message  
} AudioResult;

AudioResult audiofunction_create(double param) {
    if (param <= 0) {
        return (AudioResult){NULL, "Parameter must be positive"};
    }
    // Create object...
    return (AudioResult){object_ptr, NULL};  // NULL = success
}
```

```go
// In .go file - CGO function declarations + Go wrappers
/*
#include "native/package.m"

// Function declarations - CGO resolves AudioResult from .m file
AudioResult audiofunction_create(double param);
*/
import "C"

func Create(param float64) (unsafe.Pointer, error) {
    result := C.audiofunction_create(C.double(param))
    if result.error != nil {
        return nil, errors.New(C.GoString(result.error))
    }
    return unsafe.Pointer(result.result), nil
}
```

#### **Pattern B: Functions with Success/Fail Only**
```c
// In .m file - implementation only
const char* audiofunction_process(void* objectPtr, int param) {
    if (!objectPtr) {
        return "Object pointer is null";
    }
    // Process...
    return NULL;  // NULL = success
}
```

```go
// In .go file - CGO declaration + wrapper
/*
#include "native/package.m"

// Function declarations
const char* audiofunction_process(void* objectPtr, int param);
*/

func Process(obj unsafe.Pointer, param int) error {
    errorStr := C.audiofunction_process(obj, C.int(param))
    if errorStr != nil {
        return errors.New(C.GoString(errorStr))
    }
    return nil
}
```

---

## 📋 **Package Migration Checklist**

### **1. avaudio/format** ✅ **MIGRATION COMPLETE** 
**Status:** Successfully migrated with string-based error handling
**Architecture:** Clean CGO pattern with `native/format.m` separation

#### **✅ Completed Tasks:**
- [x] **Native file separation**: Created `avaudio/format/native/format.m`
- [x] **CGO pattern**: Direct `.m` include with function declarations in CGO block
- [x] **String-based errors**: All functions return proper error messages
- [x] **Function migrations**: All core functions converted successfully
- [x] **Test integration**: Comprehensive test documentation with correct signatures
- [x] **Error handling**: Rich, descriptive error messages throughout

#### **✅ Migrated Functions:**
- [x] `audioformat_new_mono()` → `AudioResult` with proper error handling
- [x] `audioformat_new_stereo()` → `AudioResult` with proper error handling  
- [x] `audioformat_new_with_channels()` → `AudioResult` with proper error handling
- [x] `audioformat_copy()` → `AudioResult` with proper error handling
- [x] `audioformat_get_sample_rate()` → String-based error handling
- [x] `audioformat_get_channel_count()` → String-based error handling

#### **✅ Test Documentation Added:**
- **CORRECT patterns**: `(result, error)` tuple returns documented
- **INCORRECT patterns**: Old signature mismatches clearly marked
- **Migration guide**: TestBasicFunctionality shows proper usage

---

### **2. avaudio/engine** ✅ **MIGRATION COMPLETE**
**Status:** Successfully migrated with string-based error handling + SetBufferSize leak fixed  
**Architecture:** Clean CGO pattern with `native/engine.m` separation

#### **✅ Completed Tasks:**
- [x] **Native file separation**: Uses proven CGO pattern with `native/engine.m`  
- [x] **String-based errors**: Converted from enum-based to string-based error handling
- [x] **SetBufferSize leak fixed**: Previously declared but unimplemented, now fully functional
- [x] **CGO pattern**: Direct `.m` include with function declarations working properly  
- [x] **Test integration**: Enhanced test documentation with migration guidance
- [x] **Function signature updates**: All tests updated for `(result, error)` patterns

#### **✅ Key Achievements:**
- **Fixed leaked feature**: `SetBufferSize()` now properly implemented in C with validation
- **Error pattern conversion**: From enum returns to descriptive string errors
- **Test signature fixes**: All node function calls updated for new return patterns
- **Performance validation**: Native C function calls verified working with proper logging

#### **✅ Test Documentation Added:**  
- **Migration status**: Clear documentation of COMPLETE status
- **Correct patterns**: `engine.SetBufferSize(size) → error` documented
- **Incorrect patterns**: Old enum-based returns marked for other packages to avoid

---

### **3. avaudio/node** ⚠️ **HIGH PRIORITY** 
**Current Status:** Mixed C/Go code, basic logging but no error returns
**Issues:** Functions return NULL on error but no error context

#### **Files to Create:**
- [ ] `avaudio/node/native/node.h`
- [ ] `avaudio/node/native/node.m`

#### **Functions to Convert:**
- [ ] `audionode_input_format_for_bus()` → `AudioResult audionode_input_format_for_bus()`
- [ ] `audionode_output_format_for_bus()` → `AudioResult audionode_output_format_for_bus()` 
- [ ] `audionode_number_of_inputs()` → Can return int, but add error checking
- [ ] `audionode_number_of_outputs()` → Can return int, but add error checking

#### **Error Messages to Add:**
- "Node pointer is null"
- "Invalid bus number (bus X, node has Y inputs/outputs)" 
- "No format available for bus X"
- "Node has no input/output buses"

---

### **4. avaudio/sourcenode** ⚠️ **MEDIUM PRIORITY**
**Current Status:** Complex mixed code, performance-critical audio generation
**Issues:** Audio thread code mixed with CGO, needs careful separation

#### **Files to Create:**  
- [ ] `avaudio/sourcenode/native/sourcenode.h`
- [ ] `avaudio/sourcenode/native/sourcenode.m`

#### **Functions to Convert:**
- [ ] `audiosourcenode_new()` → `AudioResult audiosourcenode_new()`
- [ ] `audiosourcenode_start()` → `const char* audiosourcenode_start()`  
- [ ] `audiosourcenode_stop()` → `const char* audiosourcenode_stop()`
- [ ] `audiosourcenode_set_frequency()` → `const char* audiosourcenode_set_frequency()`
- [ ] `audiosourcenode_set_amplitude()` → `const char* audiosourcenode_set_amplitude()`

#### **Special Considerations:**
- **Audio thread code must remain in pure ObjC** (performance critical)
- `objc_generate_sine_sample()` should stay as-is
- Error handling only for setup/control, not audio generation

#### **Error Messages to Add:**
- "Source node creation failed"
- "Invalid frequency (must be > 0)"  
- "Invalid amplitude (must be 0.0-1.0)"
- "Source node pointer is null"
- "Failed to start audio source"

---

### **5. Lower Priority Packages**

#### **avaudio/pluginchain** - **LOW PRIORITY**
- Unknown structure - needs investigation first
- May not exist yet or be experimental

#### **avaudio/tap** - **LOW PRIORITY**  
- Likely simple, follows pattern easily
- Lower impact on overall architecture

#### **avaudio/unit** - **LOW PRIORITY**
- May be wrapper around AudioUnit
- Investigate if actively used

---

## 🔧 **Step-by-Step Migration Process**

### **For Each Package:**

#### **Step 1: Analysis**
- [ ] Read through current `.go` file  
- [ ] Identify all C functions and their current error handling
- [ ] List all CGO calls and their return types
- [ ] Note any performance-critical code (audio threads)

#### **Step 2: File Structure Setup**
- [ ] Create `native/` directory
- [ ] Create `[package].h` with function declarations  
- [ ] Create `[package].m` with implementations
- [ ] Update `.go` file CGO block to include header

#### **Step 3: Error Pattern Implementation**  
- [ ] Convert pointer-returning functions to `AudioResult` pattern
- [ ] Convert success/fail functions to `const char*` pattern
- [ ] Add rich error messages for all failure cases  
- [ ] Handle NSException with @try/@catch blocks

#### **Step 4: Go Code Updates**
- [ ] Remove inline C code from Go file
- [ ] Update function calls to handle new error patterns
- [ ] Replace error checking with string-based approach
- [ ] Test compilation with `go build ./avaudio/[package]`

#### **Step 5: Testing**
- [ ] Update tests for new error return patterns
- [ ] Verify error messages are helpful and descriptive
- [ ] Test edge cases and error conditions
- [ ] Performance test for audio-critical packages

---

## 🎯 **Success Criteria**

### **For Each Migrated Package:**
- ✅ **Clean separation**: No C code in `.go` files
- ✅ **Rich errors**: Descriptive error messages, not just NULL returns  
- ✅ **Consistent pattern**: Matches `avaudio/engine` architecture
- ✅ **All tests pass**: Both compilation and runtime tests
- ✅ **Performance maintained**: No regression in audio-critical code

### **Overall Project:**  
- ✅ **Unified architecture**: All packages follow same pattern
- ✅ **Maintainable**: Clear separation of concerns
- ✅ **Developer-friendly**: Helpful error messages throughout
- ✅ **Production-ready**: Robust error handling for all failure modes

---

## 📝 **Migration Priority Order**

1. **`avaudio/format`** ✅ **COMPLETE** - Foundation package successfully migrated
2. **`avaudio/engine`** ✅ **COMPLETE** - Core functionality migrated + leaked features fixed  
3. **`avaudio/node`** - Core functionality, needed for engine integration  
4. **`avaudio/sourcenode`** - Complex but self-contained
5. **Investigate remaining packages** - Determine actual usage and priority

---

*This migration plan is based on the successful pattern established in `avaudio/engine` using string-based error handling and clean native file separation.*

---

## ⚠️ **LEAKED FEATURES STATUS**

### **avaudio/format** ✅ **COMPLETE - NO LEAKS**
- ✅ **Full functionality verified**: All functions properly call C implementations
- ✅ **String-based error handling**: All functions return proper Go errors  
- ✅ **Struct field access**: `result.error` and `result.result` work correctly
- ✅ **Native linking verified**: Tests pass, C functions are properly linked and working
- ✅ **Test integration complete**: Comprehensive documentation with correct signature patterns

### **avaudio/engine** ✅ **COMPLETE - LEAK FIXED**
- ✅ **SetBufferSize leak FIXED**: Previously declared but unimplemented, now fully functional
  - **Before**: Function only updated Go struct field, didn't call AVAudioEngine
  - **After**: Properly implements `audioengine_set_buffer_size()` in `native/engine.m`
  - **Validation**: Tests show actual C function calls with buffer size logging
- ✅ **String-based error conversion**: From enum-based to descriptive string errors
- ✅ **All functionality verified**: Tests pass, all native functions properly linked
- ✅ **Test documentation complete**: Enhanced with migration guidance and signature patterns

### **Next Migration Targets:**
1. **`avaudio/node`** - Apply proven CGO pattern, check for any leaked features
2. **`avaudio/sourcenode`** - Complex package, investigate for missing implementations  
3. **Remaining packages** - Systematic review using established migration checklist

### **Migration Pattern Success Rate:**
- **Proven CGO Pattern**: Direct `.m` include + function declarations = 100% success
- **String-based Errors**: Natural Go error integration working perfectly
- **Test Integration**: Documentation approach provides clear migration reference
- **Performance**: No regressions, native C calls verified working

**Key Achievement**: Both core packages (`format` and `engine`) now have zero leaked features and follow consistent architecture patterns ready for production use.
