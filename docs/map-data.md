# Map geometry

The `/map` page draws a world map with no client-side map library. Country
outlines are baked into `internal/service/mapsvc/geometry/geometry.go` as SVG
path data, generated from Natural Earth 110m Admin-0 Countries.

## Source and licence

- Dataset: Natural Earth 110m Admin-0 Countries
- Download: <https://www.naturalearthdata.com/downloads/110m-cultural-vectors/>
- Mirror used: <https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/ne_110m_admin_0_countries.geojson>
- Licence: public domain. Natural Earth asks for no attribution, but the source
  is recorded here and in the generated file header.

The 838 KB GeoJSON is **not committed**; `scripts/mapgen/*.geojson` is
gitignored. Only the generated Go file is committed, so the build never needs
network access.

## Regenerating

1. Download the GeoJSON to `scripts/mapgen/ne_110m_admin_0_countries.geojson`.
2. Run the generator from the repository root:

   ```sh
   go run scripts/mapgen/main.go \
     -in scripts/mapgen/ne_110m_admin_0_countries.geojson \
     -out internal/service/mapsvc/geometry/geometry.go
   ```

3. Run `go tool task check`. The output is deterministic, so regenerating from
   an unchanged dataset produces no diff.

## Projection

Equirectangular: longitude maps linearly to x and latitude to y inside an
880x440 viewBox, so the map is 2:1. The renderer adds no projection library. The
generator simplifies each ring (dedupe repeated points, drop collinear
midpoints) without changing the country set, then maps ISO 3166-1 alpha-2 codes
through `internal/domain/taxonomy`; features without a taxonomy match are
skipped.

Region presets (World, N. America, S. America, Europe, Africa, Asia, Oceania)
use viewBox rectangles in `internal/domain/taxonomy/regions.go`, in the same
880x440 coordinate space.
