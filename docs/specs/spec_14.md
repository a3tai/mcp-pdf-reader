Of course. Based on the provided "PDF Reference, third edition" for PDF 1.4, here is a structured and robust guide to the tests and functionality you would need to implement for your Go mcp server to be fully compliant. This document is distilled for a developer's perspective, focusing on *what* needs to be implemented.

### **PDF 1.4 Compliance Implementation & Test Plan**

This document outlines the core features and functionality required to build a server that is fully compliant with the Adobe PDF 1.4 specification. It is structured to follow the PDF Reference manual for easy cross-referencing.

---

### **Part 1: Foundational Concepts**

#### **1. Guiding Principles for Implementation**
*   **Device Independence:** Your server should process the PDF's content stream, which describes the page's appearance abstractly, and be capable of rendering it to various outputs. The goal is to preserve the final look regardless of the destination.
*   **Random Access:** PDF is not a sequential format. Your server must use the cross-reference table (`xref`) to locate and parse objects on demand. Processing a file from start to finish is generally incorrect.
*   **Versioning:** Your server must correctly identify the PDF version from the header (`%PDF-1.4`) and the document catalog's `Version` entry. It must handle features gracefully from earlier versions and be robust when encountering features from later versions.

---

### **Part 2: Core PDF Syntax and Structure (Chapter 3)**

This is the most critical part of your implementation. Without a compliant parser, nothing else will work.

#### **2.1. Lexical Conventions**
*   **Character Set:** Correctly handle 8-bit bytes. Differentiate between white-space, delimiter, and regular characters.
*   **White Space:** Treat any sequence of white-space characters (NUL, HT, LF, FF, CR, SP) as a single space, except within string literals.
*   **Comments:** Ignore all characters from a `%` to the next end-of-line marker.
*   **Case Sensitivity:** PDF keywords and names are case-sensitive.

#### **2.2. Object Parsing**
Your parser must be ableto read and represent the eight basic object types:
*   **Boolean:** Parse `true` and `false` keywords.
*   **Numeric:**
    *   Parse integers (e.g., `123`, `-98`).
    *   Parse real numbers (e.g., `34.5`, `-.002`).
*   **String:**
    *   **Literal Strings:** Parse `()`-delimited strings. Correctly handle balanced parentheses and backslash escape sequences (`\n`, `\r`, `\(`, `\\`, `\ddd` for octal).
    *   **Hexadecimal Strings:** Parse `<>`-delimited strings. Ignore white space and handle an odd number of hex digits by assuming a final 0.
*   **Name:** Parse `/`-prefixed objects. Handle the `#` character for hex-encoded bytes within a name (e.g., `/A#20B`).
*   **Array:** Parse `[]`-delimited sequences of any other direct PDF objects.
*   **Dictionary:** Parse `<< >>`-delimited sets of key-value pairs. Keys must be Name objects. Your server must handle duplicate keys gracefully (the value is undefined, but it shouldn't crash; typically, the last one seen is used).
*   **Stream:**
    *   Recognize a stream as a dictionary followed by the `stream` keyword, a newline, the stream bytes, and the `endstream` keyword.
    *   The stream dictionary is paramount. Your server **must** read the `Length` key to determine how many bytes of data to read.
    *   **Filters:** Implement decoding for all standard filters. This is a major area of work and requires external libraries.
        *   `ASCIIHexDecode`
        *   `ASCII85Decode`
        *   `LZWDecode`
        *   `FlateDecode` (requires a zlib-compatible library)
        *   `RunLengthDecode`
        *   `CCITTFaxDecode`
        *   `JBIG2Decode` (new in 1.4)
        *   `DCTDecode` (requires a JPEG-compatible library)
    *   Handle the `DecodeParms` dictionary to correctly configure the filters.
*   **Null:** Correctly parse the `null` object.

#### **2.3. File Structure**
*   **Header:** Verify the file starts with `%PDF-M.m`.
*   **Body:** Correctly parse sequences of indirect objects (`<obj_num> <gen_num> obj ... endobj`).
*   **Cross-Reference Table (`xref`):** This is the key to random access.
    *   Parse the `xref` keyword, subsection headers (start object, count), and the fixed 20-byte entries.
    *   Distinguish between in-use (`n`) and free (`f`) entries.
*   **Trailer:**
    *   Locate the trailer using the `startxref` value at the end of the file.
    *   Parse the `trailer` dictionary, retrieving essential keys like `Size`, `Root`, and `Prev`.
    *   **Incremental Updates:** Correctly handle files with multiple `xref` tables and trailers by reading them in reverse order using the `Prev` chain. The last definition of an object is the one that must be used.

#### **2.4. Encryption**
*   Check for the `Encrypt` dictionary in the trailer. If present, the document is encrypted.
*   Implement the **Standard Security Handler**.
*   **Password Algorithms:** Implement algorithms for validating the user and owner passwords. This involves MD5 hashing and RC4 encryption.
*   **Decryption:** Decrypt all strings and stream data using the derived encryption key. The decryption key is unique per-object, derived from the main key and the object/generation numbers.
*   **Access Permissions:** Implement checks for the permission flags (`P` entry) to control operations like printing, copying, etc.

---

### **Part 3: Document and Page Rendering (Chapters 4, 5, 6)**

#### **3.1. Graphics State**
*   Implement a graphics state machine.
*   Support the graphics state stack using the `q` (save) and `Q` (restore) operators.
*   Manage all device-independent parameters:
    *   **CTM (Current Transformation Matrix):** A 3x3 matrix for all coordinate transformations (`cm` operator).
    *   **Clipping Path:** Maintain and apply the current clipping path (`W`, `W*` operators).
    *   **Color:** Current stroking and nonstroking colors and color spaces.
    *   **Line Styles:** `line width`, `line cap`, `line join`, `miter limit`, `dash pattern`.

#### **3.2. Path Construction and Painting**
*   Maintain a "current path," which is built from segments.
*   Implement path construction operators: `m` (moveto), `l` (lineto), `c` (curveto), `h` (closepath), `re` (rectangle).
*   Implement path painting operators:
    *   `S` (stroke)
    *   `f` (fill, using nonzero winding number rule)
    *   `f*` (fill, using even-odd rule)
    *   `B`, `B*`, `b`, `b*` (fill and stroke combinations)
    *   `n` (no-op, for clipping)

#### **3.3. Color Spaces**
*   **Device Color Spaces:** `DeviceGray`, `DeviceRGB`, `DeviceCMYK`. These are fundamental.
*   **CIE-Based Color Spaces:** `CalGray`, `CalRGB`, `Lab`, and `ICCBased`. `ICCBased` requires the ability to parse ICC color profiles to correctly render colors.
*   **Special Color Spaces:**
    *   `Indexed`: Color-mapped space for reducing image data size.
    *   `Pattern`: Support for tiling patterns (replicated shapes) and shading patterns (smooth gradients). This is a complex rendering feature.
    *   `Separation` & `DeviceN`: For handling spot colors (e.g., PANTONE) and high-fidelity colors. Requires using an alternate color space and a tint transform function when the spot colorant is not available.

#### **3.4. Images and Forms**
*   **Image XObjects (`Do`):**
    *   Parse image dictionaries.
    *   Handle various `ColorSpace` and `BitsPerComponent` values.
    *   Apply image `Decode` arrays.
    *   Handle image masks (`ImageMask` = true) and explicit/color key masks (`Mask` entry).
*   **Form XObjects (`Do`):**
    *   Recursively process the form's content stream.
    *   Apply the form's `Matrix` and clip to its `BBox`.
    *   Manage the resource dictionary scope correctly.

#### **3.5. Text and Fonts**
*   Parse text objects (`BT`...`ET`).
*   Implement all text state parameters (`Tc`, `Tw`, `Tz`, `TL`, `Tf`, `Tr`, `Ts`).
*   Implement text positioning (`Td`, `TD`, `Tm`, `T*`) and showing (`Tj`, `TJ`, `'`, `"`).
*   **Font Handling:**
    *   **Standard 14 Fonts:** Your server must have built-in knowledge of these fonts (Times, Helvetica, Courier, Symbol, ZapfDingbats families).
    *   **Simple Fonts (Type 1, TrueType):** Must be able to parse their font dictionaries and font descriptors. If a font is embedded (`FontFile` or `FontFile2`), you must be able to parse that font program to get the glyph shapes.
    *   **Type 3 Fonts:** Must be able to render glyphs defined by streams of PDF graphics operators.
    *   **Composite Fonts (Type 0):** Support for CJK and large character sets. This requires parsing `CIDFont` and `CMap` (character map) objects.
    *   **ToUnicode CMap:** For reliable text extraction, implement the mapping from character codes to Unicode values defined in this optional stream.

---

### **Part 4: Advanced Graphics - Transparency (Chapter 7)**

This is the major feature of PDF 1.4. Full compliance requires implementing this model.

*   **Compositing Model:** Implement the basic compositing formula.
*   **Opacity and Shape:** Understand and apply both constant alpha (`ca`, `CA` graphics state parameters) and soft masks (`SMask` entry).
*   **Blend Modes:** Implement all standard blend modes (Normal, Multiply, Screen, Overlay, etc.).
*   **Transparency Groups:**
    *   This is the core concept. An XObject with a `Group` dictionary is a transparency group.
    *   Implement **isolated** and **knockout** group behaviors.
    *   Correctly manage the group's backdrop and blending color space.
*   **Overprinting:** Correctly simulate overprinting behavior within the transparency model, especially the `CompatibleOverprint` blend mode.

---

### **Part 5: Interactive and Interchange Features (Chapters 8, 9)**

#### **5.1. Interactive Features**
*   **Destinations:** Parse and navigate to both explicit and named destinations.
*   **Document Outline (Bookmarks):** Parse the outline hierarchy and handle the actions/destinations associated with items.
*   **Annotations:** Parse and render all standard annotation types (Text, Link, FreeText, Line, Square, Circle, Highlight, etc.).
*   **Actions:** Implement handlers for all standard action types (`GoTo`, `GoToR`, `Launch`, `URI`).
*   **Interactive Forms (AcroForms):**
    *   Parse the form field hierarchy.
    *   Handle all field types: Button (pushbuttons, checkboxes, radio buttons), Text, and Choice.
    *   Implement dynamic appearance stream generation for variable text fields.
    *   Handle form actions (`SubmitForm`, `ResetForm`, `ImportData`). This includes understanding FDF.

#### **5.2. Document Interchange**
*   **Metadata:** Be able to read both the legacy `Info` dictionary and the newer XMP `Metadata` streams.
*   **Logical Structure & Tagged PDF:**
    *   This is a cornerstone of accessibility and content reuse.
    *   Your server must be able to parse the `StructTreeRoot`.
    *   Traverse the structure element hierarchy.
    *   Distinguish real content from `Artifacts`.
    *   Understand the standard structure types (e.g., `H1`, `P`, `L`, `Table`) and their layout attributes (`Placement`, `Width`, `Height`, etc.). This is essential for features like content reflow.

---

### **Prioritization for Implementation**

Building a fully compliant server is a massive undertaking. A practical approach is to implement features in tiers:

1.  **Tier 1: Core Parsing and Display**
    *   All of Part 2 (Syntax, Objects, File Structure).
    *   Basic Graphics State (CTM, color, basic line styles).
    *   Path construction and painting.
    *   Device color spaces (`DeviceGray`, `RGB`, `CMYK`).
    *   Image XObjects with basic filters (`FlateDecode`, `DCTDecode`).
    *   Simple fonts (Standard 14, embedded TrueType/Type 1) with standard encodings.

2.  **Tier 2: Advanced Rendering and Content**
    *   All remaining color spaces (CIE-based, special).
    *   Form XObjects.
    *   Composite Fonts (Type 0, CMaps, CIDs) for CJK support.
    *   Complete text and font support (`ToUnicode`, etc.).

3.  **Tier 3: Full PDF 1.4 Compliance**
    *   All Transparency features from Part 4. This is complex and computationally intensive.
    *   All Interactive Features from Part 5 (Annotations, Actions, AcroForms).

4.  **Tier 4: High-Level Document Interchange**
    *   Logical Structure and Tagged PDF. This is less about rendering and more about understanding the document's semantic content for reuse and accessibility.
    *   Prepress features (OPI, Trapping, etc.), which are domain-specific.
