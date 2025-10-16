# School Finder TUI - Visualizations Demo

## 🎨 Visual Examples

### Search Results View

```
🏫 School Finder

Search: ╭─────────────────────────────────────────────────────────────╮
        │ high school                                                 │
        ╰─────────────────────────────────────────────────────────────╯

State Filter: CA (Ctrl+S to cycle)

Results: 25 schools | Avg Enrollment: 1,247 | Avg Teachers: 67.3

┌────────────────────────────────────────────────────────────────────────┐
│                                                                        │
│  • Lincoln High School                                                 │
│    San Francisco, CA | San Francisco USD | Students: 1,450 |          │
│    Teachers: 78.5 | 013456789012345                                    │
│                                                                        │
│  • Washington High School                                              │
│    Fremont, CA | Fremont USD | Students: 2,100 | Teachers: 95.2 |     │
│    024567890123456                                                     │
│                                                                        │
│  • Roosevelt High School                                               │
│    Los Angeles, CA | Los Angeles USD | Students: 1,850 |               │
│    Teachers: 89.0 | 035678901234567                                    │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘

Tab: Switch focus | Enter: Search/Select | Ctrl+S: Filter by state | Esc/Ctrl+C: Quit
```

---

### School Detail View (Small Elementary School)

```
🏫 School Details

╭──────────────────────────────────────────────────────────────────────╮
│  School Name:      Oakwood Elementary School                         │
│  NCESSCH ID:       123456789012                                      │
│  District:         Springfield School District                       │
│  School Type:      Regular School                                    │
│  Level:            Elementary                                        │
│  Grade Range:      K - 5                                             │
│  Charter School:   No                                                │
│  School Year:      2023-2024                                         │
╰──────────────────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────────────────╮
│  Street Address:   1234 Oak Street                                   │
│  City:             Springfield                                       │
│  State:            Massachusetts (MA)                                │
│  Zip Code:         01103                                             │
╰──────────────────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────────────────╮
│  Phone:            (413) 555-1234                                    │
│  Website:          www.oakwood.springfield.k12.ma.us                 │
╰──────────────────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────────────────╮
│  Total Enrollment: 385                                               │
│  Teachers (FTE):   22.5                                              │
│  Student/Teacher:  17.1:1                                            │
╰──────────────────────────────────────────────────────────────────────╯

📊 Metrics Visualization

Enrollment      ███████████████░░░░░░░░░░░░░░░░░░░░░░░░░ 385
Teachers (FTE)  █████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░ 22.5

Student/Teacher Ratio Analysis:
───────────◆──┃──────────┃──────────┃──────────┃───
 Excellent     Good      Average      High
Current Ratio: 17.1:1


Ctrl+Y: Copy ID to clipboard | Esc: Back to search | Ctrl+C: Quit
```

---

### School Detail View (Large High School)

```
🏫 School Details

╭──────────────────────────────────────────────────────────────────────╮
│  School Name:      Central High School                               │
│  NCESSCH ID:       987654321098                                      │
│  District:         Metro City Public Schools                         │
│  School Type:      Regular School                                    │
│  Level:            High                                              │
│  Grade Range:      9 - 12                                            │
│  Charter School:   No                                                │
│  School Year:      2023-2024                                         │
╰──────────────────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────────────────╮
│  Street Address:   5678 Main Avenue                                  │
│  City:             Metro City                                        │
│  State:            California (CA)                                   │
│  Zip Code:         90210                                             │
╰──────────────────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────────────────╮
│  Phone:            (310) 555-6789                                    │
│  Website:          www.centralhigh.metrocity.k12.ca.us               │
╰──────────────────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────────────────╮
│  Total Enrollment: 2,450                                             │
│  Teachers (FTE):   118.3                                             │
│  Student/Teacher:  20.7:1                                            │
╰──────────────────────────────────────────────────────────────────────╯

📊 Metrics Visualization

Enrollment      █████████████████████████████████████████ 2450
Teachers (FTE)  █████████████████████████████████████████ 118.3

Student/Teacher Ratio Analysis:
───┃──────────┃────────◆─┃──────────┃──────────┃───
 Excellent     Good      Average      High
Current Ratio: 20.7:1


Ctrl+Y: Copy ID to clipboard | Esc: Back to search | Ctrl+C: Quit
```

---

### School Detail View (Charter with High Ratio)

```
🏫 School Details

╭──────────────────────────────────────────────────────────────────────╮
│  School Name:      Innovation Charter Academy                        │
│  NCESSCH ID:       456789012345                                      │
│  District:         State Charter School                              │
│  School Type:      Charter School                                    │
│  Level:            Middle                                            │
│  Grade Range:      6 - 8                                             │
│  Charter School:   Yes                                               │
│  School Year:      2023-2024                                         │
╰──────────────────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────────────────╮
│  Street Address:   9012 Innovation Boulevard                         │
│  City:             Austin                                            │
│  State:            Texas (TX)                                        │
│  Zip Code:         78701                                             │
╰──────────────────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────────────────╮
│  Phone:            (512) 555-9012                                    │
│  Website:          www.innovationcharter.org                         │
╰──────────────────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────────────────╮
│  Total Enrollment: 675                                               │
│  Teachers (FTE):   21.8                                              │
│  Student/Teacher:  31.0:1                                            │
╰──────────────────────────────────────────────────────────────────────╯

📊 Metrics Visualization

Enrollment      █████████████████████████░░░░░░░░░░░░░░░ 675
Teachers (FTE)  ██████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 21.8

Student/Teacher Ratio Analysis:
───┃──────────┃──────────┃──────────┃─────────◆┃───
 Excellent     Good      Average      High
Current Ratio: 31.0:1


Ctrl+Y: Copy ID to clipboard | Esc: Back to search | Ctrl+C: Quit
```

---

## 🎯 Chart Interpretation Guide

### Bar Charts

**Length Indicates:**
- **Short bars (< 25%)**: Below average
- **Medium bars (25-75%)**: Average range
- **Long bars (> 75%)**: Above average

**Colors:**
- **Cyan (enrollment)**: No value judgment, just size
- **Magenta (teachers)**: No value judgment, just staff size

### Ratio Indicator

**Position Shows Quality:**
```
GREEN ZONE          YELLOW ZONE         RED ZONE
(Excellent/Good)    (Average)           (High/Concerning)
0-20:1              20-25:1             25+:1
```

**What the Zones Mean:**
- **Excellent (< 15:1)**: More individualized attention, smaller classes
- **Good (15-20:1)**: Solid teacher support, manageable classes
- **Average (20-25:1)**: Typical for many schools, standard support
- **High (25+:1)**: Higher class sizes, less individual attention

### Summary Statistics

**Use For:**
- Comparing schools in a search
- Understanding regional patterns
- Identifying outliers
- Context for individual schools

**Example:**
```
Results: 45 schools | Avg Enrollment: 623 | Avg Teachers: 32.4
```
If you see a school with 1,500 students, you know it's larger than the average in your search.

---

## 🌟 Visual Design Philosophy

### Colors Are Contextual
- **Green** = Low ratio (good for students)
- **Yellow** = Medium ratio (average)
- **Red** = High ratio (potential concern)

### Bars Are Relative
- Scaled to reasonable maximums (not absolute)
- 1000 students = full bar (enrollment)
- 60 teachers = full bar (teachers)

### Ratios Are Benchmarked
- Based on national averages and research
- 15:1 considered excellent (NCES data)
- 25:1 considered high (class size research)

---

## 💡 Tips for Using Visualizations

1. **Look for patterns**: Search a district to compare schools visually
2. **Green is good**: For ratios, greener is better
3. **Context matters**: Small schools naturally have different profiles
4. **Use with data**: Charts supplement numbers, don't replace them
5. **State differences**: Different states have different typical ratios

---

## 🔮 Future Visualization Ideas

- Grade-by-grade enrollment bars
- Demographic distribution charts
- Historical trend sparklines
- Multi-school comparison view
- Geographic heat maps (if coordinate data added)
- Performance metric gauges
- Resource allocation pie charts
