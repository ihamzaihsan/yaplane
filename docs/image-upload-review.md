# Image-upload extension review

Reviewed against the supplied image-upload objective and both audit checklists. The upload implementation is kept separate from the security extension.

## Verified requirements

- An authenticated user can create an image-and-text discussion with a PNG, JPEG, or GIF attachment.
- The image path is stored with the post and the file is written under `static/uploads/`.
- Guests can read the discussion and retrieve the same image bytes after navigating back to the post.
- An upload larger than 20 MiB is rejected with the existing visible size warning.
- The implementation uses Go, SQLite, and the allowed packages; no upload framework is needed.

These requirements were checked with generated images and an isolated database before repository cleanup. The automated test files are retained in Git history.

## Remaining limitations

The basic upload flow works, but the implementation does not fully satisfy the intended image-size boundary or all error-handling expectations:

1. The 20 MiB cap applies to the entire multipart request, including text and form overhead. Therefore an image of exactly 20 MiB can be rejected even though the subject allows it. The server should check the image size separately and allow bounded multipart overhead.
2. The handler saves an attachment before validating title, content, and categories. A rejected post or failed database write can leave an unused file behind.
3. Some upload errors are ignored, including non-missing `FormFile` errors and the rewind result. Failed copies can leave partial files. MIME sniffing checks the header; it does not verify that the full image is decodable.

The security extension fixes a separate serving issue: a valid image uploaded with an `.html` filename is now served as its detected image MIME type with `nosniff`, rather than relying on the extension. This preserves image viewing while preventing the upload from being interpreted as an HTML page. Existing images are retained.

The conclusion is **working core functionality with specific compliance and cleanup gaps**, rather than a claim that every audit item already passes. No image-size or file-cleanup refactor is bundled into the security extension.
