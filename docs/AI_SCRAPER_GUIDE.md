# AI Website Scraper - User Guide

## Overview

The School Finder TUI now includes an AI-powered website scraper that uses **Claude 3.5 Haiku** to extract structured information from school websites automatically.

## Features

### 🤖 What It Does

- **Fetches** the school's website HTML
- **Analyzes** content using Claude 3.5 Haiku (fast, cost-effective)
- **Extracts** structured data including:
  - Principal and administration
  - Mascot and school colors
  - AP courses and special programs
  - Sports teams and clubs
  - Facilities and achievements
  - Mission statement
  - And much more!
- **Caches** results locally (30-day cache)
- **Displays** formatted information in the TUI

### 💾 Caching System

- Scraped data is saved to `.school_cache/` directory
- Cached for 30 days to avoid redundant API calls
- JSON format for easy inspection and backup
- Automatic cache invalidation after 30 days

## Setup

### 1. Get an Anthropic API Key

1. Sign up at [console.anthropic.com](https://console.anthropic.com)
2. Generate an API key
3. Copy the key (starts with `sk-ant-...`)

### 2. Set Environment Variable

**Linux/Mac:**
```bash
export ANTHROPIC_API_KEY='sk-ant-your-key-here'
```

**Or add to your `~/.bashrc` or `~/.zshrc`:**
```bash
echo 'export ANTHROPIC_API_KEY="sk-ant-your-key-here"' >> ~/.bashrc
source ~/.bashrc
```

**Windows (PowerShell):**
```powershell
$env:ANTHROPIC_API_KEY='sk-ant-your-key-here'
```

**Windows (CMD):**
```cmd
set ANTHROPIC_API_KEY=sk-ant-your-key-here
```

### 3. Run the Application

```bash
./schoolfinder
```

The AI scraper will be automatically enabled if the API key is found.

## Usage

### In the TUI

1. Search for and select a school
2. Press **Ctrl+A** to scrape the website
3. Wait for Claude to analyze (~3-10 seconds)
4. View the extracted information below the main details

### Visual Indicators

**While scraping:**
```
⏳ Scraping website with AI...
```

**After successful extraction:**
```
🤖 AI-Extracted Information
Source: https://school.edu
Extracted: 2025-10-16 00:15
```

**If no website available:**
The Ctrl+A option won't appear in the help text.

## Extracted Data Fields

### Leadership
- **Principal**: Name of principal/headmaster
- **Vice Principals**: List of assistant principals

### Identity
- **Mascot**: School mascot name
- **School Colors**: Official colors
- **Founded**: Year established

### Academics
- **AP Courses**: Advanced Placement offerings
- **Honors Programs**: Honors track information
- **Special Programs**: IB, STEM, magnet programs, etc.
- **Languages**: Foreign languages offered

### Activities
- **Sports**: Athletic teams and programs
- **Clubs**: Student organizations
- **Arts**: Visual and performing arts programs

### Facilities
- Notable buildings, labs, auditoriums, etc.

### Schedule
- **Bell Schedule**: Class period structure
- **School Hours**: Start and end times

### Recognition
- **Achievements**: Awards, rankings, recognitions
- **Accreditations**: Official accreditations

### Mission
- School mission statement or motto

## Example Output

```
🤖 AI-Extracted Information
Source: https://lincolnhigh.example.edu
Extracted: 2025-10-16 00:15

Principal: Dr. Sarah Johnson
Vice Principals: Mr. Tom Williams, Ms. Lisa Chen

School Identity:
  Mascot: Lions
  Colors: Blue, Gold

AP Courses:
  • AP Calculus AB/BC
  • AP Physics C
  • AP English Literature
  • AP US History
  • AP Biology
  • AP Computer Science

Special Programs:
  • International Baccalaureate (IB)
  • STEM Academy
  • College Dual Enrollment

Sports: Football, Basketball, Soccer, Track & Field, Swimming,
        Volleyball, Baseball, Softball, Tennis, Cross Country

Clubs: 45 clubs available

School Hours: 8:00 AM - 3:15 PM

Mission: Empowering students to become lifelong learners and
         responsible global citizens through rigorous academics
         and character development.

Achievements:
  • National Blue Ribbon School (2023)
  • California Distinguished School
  • Gold Medal - US News Rankings
```

## Cost Considerations

### Pricing (Claude 3.5 Haiku)
- **Input**: $0.80 per million tokens (~$0.0008 per page)
- **Output**: $4.00 per million tokens (~$0.002 per extraction)

**Estimated cost per school:** $0.001 - $0.005 (less than half a cent!)

### Cost-Saving Features
1. **30-day cache** - Avoid re-scraping same schools
2. **Haiku model** - 8-10x cheaper than Sonnet/Opus
3. **Smart truncation** - Limits content to 100KB
4. **Fast processing** - 3-5 second responses

### Budget Management
- **1,000 schools**: ~$1-5
- **10,000 schools**: ~$10-50
- Free tier: Check Anthropic's current offerings

## Troubleshooting

### "No website available for this school"
- School doesn't have a website in the database
- Website field is empty or null
- Check the Contact section to confirm

### "Failed to fetch website"
- Website URL is incorrect or broken
- Network connectivity issues
- Website blocks automated requests
- Try visiting the URL manually to verify

### "Claude API error"
- Invalid API key
- API key expired or deactivated
- Rate limit exceeded (rare with Haiku)
- Check your Anthropic console

### "Failed to parse Claude response"
- Website has unusual formatting
- Very minimal content on website
- The JSON extraction failed
- Check cached file for details

### Empty/Missing Fields
- Website doesn't contain that information
- Information in non-standard format
- Claude couldn't locate the data
- Some schools have minimal websites

## Cache Management

### View Cache
```bash
ls -lh .school_cache/
```

### Check a Cached School
```bash
cat .school_cache/123456789012.json | jq .
```

### Clear Cache
```bash
rm -rf .school_cache/
```

### Clear Old Cache (30+ days)
```bash
find .school_cache/ -name "*.json" -mtime +30 -delete
```

## Privacy & Ethics

### What We Do
✅ Publicly accessible information only
✅ Respects robots.txt (when appropriate)
✅ User-Agent identifies as educational tool
✅ Caches to minimize requests
✅ Used for research and informational purposes

### What We Don't Do
❌ No login/authentication bypass
❌ No personal data extraction
❌ No excessive scraping
❌ No data resale

### Responsible Use
- Use for educational/research purposes
- Don't abuse the scraping feature
- Respect website terms of service
- Cache reduces server load

## Advanced Features

### Custom Cache Directory
Modify `main.go` to change cache location:
```go
aiScraper, err = NewAIScraperService(apiKey, "/path/to/cache")
```

### Adjusting Cache Duration
Modify `ai_scraper.go` line ~208:
```go
if time.Since(cached.ExtractedAt) < 30*24*time.Hour {
```

### Increasing Content Limit
Modify `ai_scraper.go` line ~114:
```go
if len(content) > 100000 {  // Increase this value
```

### Custom Extraction Fields
Modify the prompt in `ExtractSchoolData()` to request additional fields.

## API Limits

### Anthropic Rate Limits (as of 2025)
- **Haiku**: Very high limits
- **Free tier**: Check current limits
- **Paid tier**: Virtually unlimited for this use case

### Typical Usage Patterns
- Interactive use: 1-10 schools/session
- Batch use: Can scrape hundreds efficiently
- Cache dramatically reduces API calls

## Keyboard Shortcuts

**In Detail View:**
- `Ctrl+A` - Scrape website with AI
- `Ctrl+Y` - Copy school ID to clipboard
- `Esc` - Return to search
- `Ctrl+C` - Quit application

## Tips & Best Practices

1. **Check cache first** - Data is reused for 30 days
2. **Wait patiently** - AI extraction takes 3-10 seconds
3. **Verify accuracy** - AI extractions should be verified
4. **Report issues** - Some websites are harder to parse
5. **Use responsibly** - Don't abuse the feature

## Future Enhancements

Potential features:
- [ ] Multiple website page analysis
- [ ] Historical data tracking
- [ ] Comparison with other schools
- [ ] Export to structured formats
- [ ] Confidence scores for extracted data
- [ ] Website screenshot capture
- [ ] Social media integration
- [ ] Parent reviews aggregation

## Support

### Issues
- Invalid extractions
- API errors
- Caching problems

### Improvements
- Better data extraction
- Additional fields
- UI/UX enhancements

## Legal

This tool is for educational and research purposes. Users are responsible for:
- Complying with website terms of service
- Respecting rate limits
- Using data ethically
- Following local laws regarding web scraping

School website data is publicly available information. This tool simply automates the process of reading and organizing that public information.
