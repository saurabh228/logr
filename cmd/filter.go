package cmd

// filter.go intentionally left minimal — the filter behaviour lives in
// root.go's RunE so that `logr` (with no subcommand) reads stdin and filters.
//
// If you want to expose the filter as an explicit subcommand in the future,
// wire it up here and delegate to the same runFilter function.
