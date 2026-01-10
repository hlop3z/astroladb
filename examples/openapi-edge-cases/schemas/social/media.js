// Edge case: Polymorphic attachment (media can belong to post, comment, or profile)
export default table({
  id: col.id(),
  // Who uploaded
  uploaded_by: col.belongs_to("auth.user"),
  // Polymorphic: what this media is attached to
  attachable: col.belongs_to_any([
    "content.post",
    "content.comment",
    "auth.profile",
  ]),
  filename: col.name(),
  url: col.url(),
  mime_type: col.string(100),
  size_bytes: col.quantity(),
  width: col.integer().optional().docs("Width in pixels for images/videos"),
  height: col.integer().optional().docs("Height in pixels for images/videos"),
  file_type: col.enum(["image", "video", "audio", "document"]).default("image"),
  sort_order: col.counter(),
}).timestamps();
