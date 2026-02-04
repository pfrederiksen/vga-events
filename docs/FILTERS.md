# Event Filtering System

## Overview

The VGA Events Bot now supports powerful event filtering capabilities, allowing users to narrow down events by date, course, city, and weekends. Filters can be saved as presets for quick reuse.

## Features

### Filter Criteria

1. **Date Range**: Filter events by specific date ranges
   - Single month: "March"
   - Date range within month: "Mar 1-15"
   - Date range across months: "March 1 - April 15"

2. **Course Name**: Filter by golf course (case-insensitive substring match)
   - Example: "Pebble Beach", "Shadow Creek"

3. **City**: Filter by city name (case-insensitive substring match)
   - Example: "Las Vegas", "San Diego"

4. **Weekends Only**: Toggle to show only Saturday/Sunday events

### Combining Filters

Filters can be combined for powerful queries:
```
/filter date "Mar 1-15"
/filter weekends
/filter course "Pebble Beach"
```

This creates a filter for: "Pebble Beach events on weekends in March 1-15"

### Saving Filters

Save frequently-used filter combinations as named presets:
```
/filter save "My Weekend Events"
```

Load saved filters instantly:
```
/filter load "My Weekend Events"
```

## Bot Commands

### `/filter` - Main Filter Command

Show current filter status:
```
/filter
```

Set date range filter:
```
/filter date "Mar 1-15"
/filter date "March 1 - April 15"
/filter date "March"
```

Add course filter:
```
/filter course "Pebble Beach"
```

Add city filter:
```
/filter city "Las Vegas"
```

Toggle weekends-only:
```
/filter weekends
```

Save current filter:
```
/filter save "March Weekends"
```

Load saved filter:
```
/filter load "March Weekends"
```

Delete saved filter:
```
/filter delete "March Weekends"
```

Clear active filter:
```
/filter clear
```

### `/filters` - List Saved Filters

View all saved filter presets:
```
/filters
```

Shows:
- Filter name
- Filter criteria
- Active filter indicator (✅)

## Implementation Details

### Package Structure

```
internal/filter/
├── filter.go       - Core filter types and matching logic
├── filter_test.go  - Filter tests (83.8% coverage)
├── parser.go       - Date range parsing
└── parser_test.go  - Parser tests
```

### Core Types

**Filter**: Represents filtering criteria
```go
type Filter struct {
    DateFrom     *time.Time
    DateTo       *time.Time
    Courses      []string
    WeekendsOnly bool
    States       []string
    Cities       []string
    MaxPrice     float64
}
```

**FilterPreset**: Named filter configuration
```go
type FilterPreset struct {
    Name      string
    Filter    *Filter
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### UserPreferences Integration

Added to `UserPreferences`:
```go
SavedFilters map[string]*FilterPreset  // name → filter preset
ActiveFilter string                      // name of active filter
```

Helper methods:
- `SaveFilter(name, filter)` - Save/update a filter
- `GetFilter(name)` - Retrieve saved filter
- `DeleteFilter(name)` - Remove saved filter
- `GetActiveFilter()` - Get currently active filter
- `ApplyFiltersToEvents(events)` - Apply active filter to events

### Date Parsing

Supports multiple formats:
- `"Mar 1-15"` → March 1 to March 15
- `"March 1 - April 15"` → March 1 to April 15
- `"March"` → Entire month of March
- `"Dec 25 - Jan 5"` → Cross-year range (Dec 25 to Jan 5 next year)

Smart year handling:
- If month is in the past, assumes next year
- Cross-year ranges automatically handled

### Event Filtering

Filters are applied in `/events`, `/search`, and `/near` commands:

1. **State filtering**: Events from subscribed states only
2. **Past event filtering**: Hide past events (if enabled)
3. **Active filter**: Apply user's custom filter
4. **Results**: Filtered events with active filter indicator

## Usage Examples

### Weekend Events in March

```
/filter date "March"
/filter weekends
/filter save "March Weekends"
```

### Pebble Beach Events

```
/filter course "Pebble Beach"
/filter save "Pebble Beach"
```

### Las Vegas Weekend Events

```
/filter city "Las Vegas"
/filter weekends
/filter save "Vegas Weekends"
```

### Spring Events (Multi-Month)

```
/filter date "March 1 - May 31"
/filter save "Spring Events"
```

## Testing

Filter package has comprehensive test coverage (83.8%):

- Filter matching logic
- Filter application to event lists
- Date range parsing
- Filter cloning
- Filter string representation
- Parser edge cases

Run tests:
```bash
go test ./internal/filter/... -v -cover
```

## Future Enhancements

Potential improvements:

1. **Price Filtering**: When price data becomes available
   - `/filter price 100` - Max price $100

2. **State Filtering**: Within subscribed states
   - `/filter state CA` - Only California events

3. **Complex Queries**: Boolean operators
   - `/filter course "Pebble OR Augusta"`

4. **Quick Filters**: Predefined shortcuts
   - `/filter upcoming-week` - Next 7 days
   - `/filter this-month` - Current month

5. **Filter Sharing**: Share filters with friends
   - `/filter share "Vegas Weekends" @friend`

## Related Documentation

- [CLAUDE.md](../CLAUDE.md) - Development workflow
- [README.md](../README.md) - Project overview
- [Event Types](../internal/event/event.go) - Event data structure
