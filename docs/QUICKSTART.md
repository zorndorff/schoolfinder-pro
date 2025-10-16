# Quick Start Guide

Get up and running with School Finder TUI in 5 minutes!

## Step 1: Build the Application

```bash
cd /home/zac/projects/sketchpad/tmpdata/schools
go build -o schoolfinder
```

## Step 2: Run It!

```bash
./schoolfinder
```

**First Time Setup:** If you don't have the data files yet, the application will automatically prompt you to download them:

```
⚠️  Missing required data files:
   - ccd_sch_029_2324_w_1a_073124.csv
   - ccd_sch_059_2324_l_1a_073124.csv
   - ccd_sch_052_2324_l_1a_073124.csv

Would you like to download them now? (y/N): y
```

The application will download and extract the files automatically (several hundred MB, may take a few minutes).

That's it! You can now search schools.

## Step 3: Try a Search

1. Type a school name, city, or state: `lincoln high`
2. Press `Enter` to search
3. Use arrow keys to browse results
4. Press `Enter` again to view details

## Step 4: Enable AI Features (Optional)

```bash
# Get API key from https://console.anthropic.com
export ANTHROPIC_API_KEY='sk-ant-your-key-here'

# Run again
./schoolfinder
```

Now when viewing a school, press `Ctrl+A` to extract website data!

## Basic Controls

**Searching:**
- Type to search
- `Tab` to switch between input and results
- `Ctrl+S` to cycle state filters

**Viewing:**
- `Enter` to view school details
- `Ctrl+A` to scrape website (if API key set)
- `Ctrl+Y` to copy school ID
- `Esc` to go back

**Quit:**
- `Ctrl+C` or `Esc` (from search view)

## What You'll See

### Search View
```
🏫 School Finder

Search: [your search here]
State Filter: All States (Ctrl+S to cycle)

Results: 25 schools | Avg Enrollment: 1,247 | Avg Teachers: 67.3

• Lincoln High School
  San Francisco, CA | SFUSD | Students: 1,450 | Teachers: 78.5
```

### Detail View
```
🏫 School Details

School Name:         Lincoln High School
NCESSCH ID:          060123456789
District:            San Francisco Unified School District
Grade Range:         9 - 12
Total Enrollment:    1,450
Teachers (FTE):      78.5
Student/Teacher:     18.5:1

📊 Metrics Visualization
[Bar charts showing enrollment and teacher counts]
[Visual ratio indicator]

🤖 AI-Extracted Information (after Ctrl+A)
[Principal, mascot, AP courses, sports, clubs, etc.]
```

## Example Searches

Try these:
- `elementary` - Find elementary schools
- `charter` - Find charter schools
- `94102` - Search by ZIP code
- `Los Angeles` - Search by city
- Then use `Ctrl+S` to filter by state!

## Tips

1. **Faster searching**: Use state filter (`Ctrl+S`) to narrow results
2. **AI extractions**: Press `Ctrl+A` in detail view (requires API key)
3. **Copy IDs**: Use `Ctrl+Y` to copy school ID to clipboard
4. **Cached data**: AI extractions are cached for 30 days

## Troubleshooting

**Missing data files?**
- The app will automatically prompt you to download them on first run
- If you declined, run the app again and choose 'y' when prompted
- Files are downloaded from NCES (National Center for Education Statistics)
- Total size: ~several hundred MB

**"No schools found"**
- Try a broader search term
- Check spelling
- Try searching by city or state instead

**"No website available"**
- School doesn't have website in database
- AI scraper won't be available

**AI not working?**
- Did you set `ANTHROPIC_API_KEY`?
- Check: `echo $ANTHROPIC_API_KEY`
- Get key: https://console.anthropic.com

## Next Steps

- Read [README.md](README.md) for full documentation
- See [AI_SCRAPER_GUIDE.md](AI_SCRAPER_GUIDE.md) for AI features
- Check [CHARTS.md](CHARTS.md) to understand visualizations
- View [FEATURE_SUMMARY.md](FEATURE_SUMMARY.md) for complete features

## Need Help?

Check the documentation files or the help text shown at the bottom of each view.

---

Happy school exploring! 🎓
