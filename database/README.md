# Database Cleanup Log for Anna's Archive Spotify Dataset

This document outlines the cleanup process applied to the original Spotify metadata database from **Anna's Archive** to create the compacted version used in this project. For full details on the original dataset, please refer to the [Anna's Archive blog post](https://annas-archive.li/blog/2025-12-20_backing-up-spotify).

## Milestones

- Removed all tracks in metadata with popularity < 30. <!-- reduce low-signal items -->
- Dropped the metadata `available_markets` table and cleaned foreign keys. <!-- schema simplification -->
- Removed track URLs that no longer exist in metadata. <!-- keep cross-db consistency -->
- Dropped `playlists.collaborative` and `playlists.public` columns. <!-- removed unused flags -->
- Dropped `playlist_images.width` and `playlist_images.height` columns. <!-- dimensions no longer stored -->
- Removed playlists with followers_total <= 5. <!-- filter out tiny playlists -->

Note: No new fields were added; only deletions. The full original database remains usable. <!-- compatibility note -->

## Data Distribution Notes

**Detailed distribution**

- followers = 0: 2.73 million (41.38%). <!-- zero-follower playlists -->
- followers = 1: 0.96 million (14.53%). <!-- single-follower playlists -->
- followers = 2: 0.47 million (7.21%). <!-- two followers -->
- followers 3-5: 0.59 million (9.06%). <!-- small-range bucket -->
- followers 6-10: 0.32 million (4.92%). <!-- small-range bucket -->
- followers 11-20: 0.22 million (3.36%). <!-- small-range bucket -->
- followers > 1000: 0.47 million (7.13%). <!-- high-follower bucket -->
