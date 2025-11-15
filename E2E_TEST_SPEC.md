# End-to-End Test Specification: BYOD (Bring Your Own Data) Feature

## Overview
This document specifies comprehensive end-to-end tests for the BYOD feature that allows users to import custom CSV files into the School Finder application and query them using the AI Data Explorer.

## Test Environment Setup

### Prerequisites
- School Finder application running on localhost:8080
- ANTHROPIC_API_KEY environment variable set
- Sample CSV files for testing (see Test Data section)
- tmpdata/data.duckdb initialized with school data

### Test Data Files

#### 1. Valid CSV - Student Grades (test_grades.csv)
```csv
student_id,student_name,grade_level,subject,score,test_date
1001,Alice Johnson,9,Math,92,2024-01-15
1002,Bob Smith,9,Math,88,2024-01-15
1003,Carol White,10,English,95,2024-01-15
1004,David Brown,10,English,87,2024-01-15
1005,Eve Davis,11,Science,91,2024-01-15
```

#### 2. Valid CSV - Teacher Assignments (test_teachers.csv)
```csv
teacher_id,teacher_name,department,years_experience,certification
T001,John Doe,Mathematics,15,Advanced
T002,Jane Smith,English,8,Standard
T003,Mike Johnson,Science,12,Advanced
T004,Sarah Williams,History,5,Standard
```

#### 3. Invalid CSV - Missing Headers (test_invalid.csv)
```csv
1001,Alice,92
1002,Bob,88
```

#### 4. Large CSV - Performance Test (test_large.csv)
Generate a CSV with 50,000 rows and 10 columns for performance testing.

---

## Test Suite 1: Import Page Access and UI

### TEST-001: Access Import Page
**Objective:** Verify that the import page loads correctly

**Steps:**
1. Navigate to `http://localhost:8080`
2. Click on "Import Data" in the navigation menu

**Expected Results:**
- ✓ Import page loads successfully (HTTP 200)
- ✓ Page title shows "Import Data - School Finder"
- ✓ Navigation shows "Import Data" as active
- ✓ Form contains CSV file upload field
- ✓ Form contains table name input field
- ✓ Form contains description textarea
- ✓ "Import Data" button is present
- ✓ "How it works" section with 4 steps is visible

---

## Test Suite 2: Form Validation

### TEST-002: Submit Empty Form
**Objective:** Verify form validation for required fields

**Steps:**
1. Navigate to import page
2. Click "Import Data" button without filling any fields

**Expected Results:**
- ✓ Browser shows HTML5 validation error for "CSV File" field
- ✓ Form does not submit
- ✓ No network request is made

### TEST-003: Submit Without CSV File
**Objective:** Verify CSV file is required

**Steps:**
1. Navigate to import page
2. Enter table name: "test_table"
3. Enter description: "Test data"
4. Click "Import Data" without selecting a file

**Expected Results:**
- ✓ Browser shows HTML5 validation error for file input
- ✓ Form does not submit

### TEST-004: Submit Without Table Name
**Objective:** Verify table name is required

**Steps:**
1. Navigate to import page
2. Upload test_grades.csv
3. Enter description: "Student grade data"
4. Leave table name empty
5. Click "Import Data"

**Expected Results:**
- ✓ Browser shows HTML5 validation error for table name
- ✓ Form does not submit

### TEST-005: Submit Without Description
**Objective:** Verify description is required

**Steps:**
1. Navigate to import page
2. Upload test_grades.csv
3. Enter table name: "student_grades"
4. Leave description empty
5. Click "Import Data"

**Expected Results:**
- ✓ Browser shows HTML5 validation error for description
- ✓ Form does not submit

### TEST-006: Invalid Table Name Format
**Objective:** Verify table name validation pattern

**Steps:**
1. Navigate to import page
2. Upload test_grades.csv
3. Try table names: "123invalid", "Invalid-Name", "UPPERCASE"
4. Enter description: "Test"
5. Attempt to submit

**Expected Results:**
- ✓ Browser shows validation error for table names starting with numbers
- ✓ Browser shows validation error for table names with hyphens
- ✓ Browser accepts lowercase with underscores only

---

## Test Suite 3: Successful Import Workflow

### TEST-007: Import Small CSV Successfully
**Objective:** Complete full import workflow with valid data

**Steps:**
1. Navigate to import page
2. Upload test_grades.csv
3. Enter table name: "student_grades"
4. Enter description: "Student grade data from Q1 2024. Contains test scores by subject."
5. Click "Import Data"
6. Wait for processing to complete

**Expected Results:**
- ✓ Loading indicator appears with "Processing your data..." message
- ✓ Processing completes within 10 seconds
- ✓ Success message "Import Successful!" is displayed
- ✓ Summary shows correct table name: "student_grades"
- ✓ Summary shows correct row count: 5
- ✓ Summary shows correct column count: 6
- ✓ Data metrics table displays for all 6 columns
- ✓ AI description is generated and displayed
- ✓ Processing log shows all stages:
  - Parse Form
  - File Upload
  - Save File
  - Analyze Data
  - Create Table
  - Generate Descriptions
  - Add Comments
- ✓ "Query Your Data" button links to /agent
- ✓ "Import More Data" button reloads the page

### TEST-008: Verify File Saved to Disk
**Objective:** Confirm CSV is saved to user_data directory

**Steps:**
1. Complete TEST-007
2. Check filesystem at tmpdata/user_data/

**Expected Results:**
- ✓ Directory tmpdata/user_data/ exists
- ✓ File student_grades.csv exists in directory
- ✓ File size matches uploaded file

### TEST-009: Verify Table Created in Database
**Objective:** Confirm table is created in DuckDB

**Steps:**
1. Complete TEST-007
2. Connect to tmpdata/data.duckdb
3. Run: `SHOW ALL TABLES;`
4. Run: `SELECT * FROM student_grades LIMIT 5;`

**Expected Results:**
- ✓ Table "student_grades" appears in table list
- ✓ Query returns 5 rows
- ✓ All 6 columns are present
- ✓ Data types are correctly inferred

### TEST-010: Verify Table and Column Comments
**Objective:** Confirm AI-generated comments are stored

**Steps:**
1. Complete TEST-007
2. Run: `SELECT comment FROM duckdb_tables() WHERE table_name = 'student_grades';`
3. Run: `SELECT column_name, comment FROM duckdb_columns() WHERE table_name = 'student_grades';`

**Expected Results:**
- ✓ Table comment contains AI-generated description
- ✓ All columns have AI-generated comments
- ✓ Comments are relevant to the data

---

## Test Suite 4: Data Metrics and Analysis

### TEST-011: Verify SUMMARIZE Metrics
**Objective:** Confirm DuckDB SUMMARIZE provides correct metrics

**Steps:**
1. Import test_grades.csv as "student_grades"
2. Examine metrics table in results

**Expected Results:**
For each column, metrics table shows:
- ✓ Column Name
- ✓ Column Type (e.g., BIGINT for student_id, VARCHAR for student_name)
- ✓ Min value (e.g., "1001" for student_id)
- ✓ Max value (e.g., "1005" for student_id)
- ✓ Unique count
- ✓ Null percentage (0% for all in test data)

### TEST-012: AI Description Quality
**Objective:** Verify AI generates meaningful descriptions

**Steps:**
1. Import test_teachers.csv as "teacher_roster"
2. Description: "Teacher assignments and certifications for the district"
3. Review AI-generated table and column descriptions

**Expected Results:**
- ✓ Table description mentions teachers/assignments
- ✓ Table description mentions how to use in queries
- ✓ Column "teacher_id" comment explains it's an identifier
- ✓ Column "years_experience" comment mentions experience level
- ✓ Column "certification" comment explains certification types

---

## Test Suite 5: AI Data Explorer Integration

### TEST-013: Discover Imported Table via Schema Tool
**Objective:** Verify AI agent can discover user-imported tables

**Steps:**
1. Import test_grades.csv as "student_grades"
2. Navigate to /agent
3. Enter query: "What tables are available?"
4. Submit query

**Expected Results:**
- ✓ AI agent uses 'schema' tool
- ✓ Response lists "student_grades" among tables
- ✓ Response shows table has 6 columns
- ✓ Response includes table comment if available

### TEST-014: Query Imported Data
**Objective:** Verify AI agent can query user-imported table

**Steps:**
1. Import test_grades.csv as "student_grades"
2. Navigate to /agent
3. Enter query: "What is the average score for each subject in student_grades?"
4. Submit query

**Expected Results:**
- ✓ AI generates SQL using student_grades table
- ✓ SQL includes GROUP BY subject
- ✓ SQL includes AVG(score)
- ✓ Query executes successfully
- ✓ Response shows average scores for Math, English, Science
- ✓ Results are formatted clearly

### TEST-015: Join Imported Data with School Data
**Objective:** Verify AI can join user data with existing school tables

**Steps:**
1. Import CSV with NCESSCH column matching school IDs
2. Navigate to /agent
3. Enter query: "Join my imported data with the directory table"

**Expected Results:**
- ✓ AI generates SQL with proper JOIN
- ✓ Query executes successfully
- ✓ Results combine both datasets

### TEST-016: Schema Tool Shows Comments
**Objective:** Verify schema tool returns table and column comments

**Steps:**
1. Import test_grades.csv as "student_grades"
2. Navigate to /agent
3. Enter query: "Show me the schema for student_grades table"
4. Submit query

**Expected Results:**
- ✓ AI calls schema tool
- ✓ Schema response includes table_comment field
- ✓ Schema response includes comment field for each column
- ✓ AI incorporates comments into explanation

---

## Test Suite 6: Error Handling

### TEST-017: Import Duplicate Table Name
**Objective:** Verify error when table already exists

**Steps:**
1. Import test_grades.csv as "student_grades"
2. Wait for success
3. Try to import test_teachers.csv also as "student_grades"

**Expected Results:**
- ✓ Import fails with error message
- ✓ Error message indicates table already exists
- ✓ Error message suggests using different table name
- ✓ Original table remains intact

### TEST-018: Import Invalid CSV Format
**Objective:** Verify error handling for malformed CSV

**Steps:**
1. Upload test_invalid.csv (no headers)
2. Table name: "invalid_test"
3. Description: "Test invalid data"
4. Submit

**Expected Results:**
- ✓ Import fails during analysis or creation stage
- ✓ Error message is clear and helpful
- ✓ User can try again with different file

### TEST-019: Import File Too Large
**Objective:** Verify file size limit enforcement

**Steps:**
1. Attempt to upload a 150MB CSV file
2. Observe behavior

**Expected Results:**
- ✓ Upload rejected with clear error message
- ✓ Message mentions 100MB limit
- ✓ User can select different file

### TEST-020: AI Description Generation Fails
**Objective:** Verify graceful handling when AI is unavailable

**Steps:**
1. Temporarily unset ANTHROPIC_API_KEY
2. Attempt CSV import
3. Observe results

**Expected Results:**
- ✓ Import completes successfully
- ✓ Table is created
- ✓ AI description shows "AI description generation failed"
- ✓ Table and column comments are not added (or show defaults)
- ✓ User can still query the table

---

## Test Suite 7: Performance Tests

### TEST-021: Import Large File
**Objective:** Verify performance with large dataset

**Steps:**
1. Generate or use test_large.csv (50,000 rows, 10 columns)
2. Import as "large_dataset"
3. Monitor processing time

**Expected Results:**
- ✓ Import completes within 30 seconds
- ✓ All processing stages complete
- ✓ Memory usage remains reasonable (<500MB)
- ✓ UI remains responsive
- ✓ Success message displays with correct row count

### TEST-022: Query Large Imported Table
**Objective:** Verify AI agent performs well with large tables

**Steps:**
1. Import test_large.csv
2. Navigate to /agent
3. Query: "What is the total count in large_dataset?"

**Expected Results:**
- ✓ Query returns within 5 seconds
- ✓ Result is accurate
- ✓ No timeout errors

### TEST-023: Multiple Concurrent Imports
**Objective:** Test behavior with multiple users importing simultaneously

**Steps:**
1. Open 3 browser tabs
2. Start imports in all tabs simultaneously
3. Monitor completion

**Expected Results:**
- ✓ All imports complete successfully
- ✓ No database locks or conflicts
- ✓ Each table is created correctly

---

## Test Suite 8: UI/UX Tests

### TEST-024: Progress Indicator Behavior
**Objective:** Verify loading states during import

**Steps:**
1. Start import process
2. Observe UI changes

**Expected Results:**
- ✓ Loading spinner appears immediately
- ✓ "Processing your data..." message is visible
- ✓ Form is disabled during processing
- ✓ Progress indicator disappears when complete

### TEST-025: Processing Log Details
**Objective:** Verify processing log provides useful information

**Steps:**
1. Complete a successful import
2. Expand "View Processing Log" details

**Expected Results:**
- ✓ All stages are listed in order
- ✓ Each stage shows duration
- ✓ File size is shown
- ✓ Row and column counts are shown
- ✓ Durations are reasonable

### TEST-026: Responsive Design
**Objective:** Verify import page works on mobile

**Steps:**
1. Resize browser to 375px width (mobile)
2. Test import workflow

**Expected Results:**
- ✓ Form is usable on small screen
- ✓ File upload works
- ✓ Buttons are tappable
- ✓ Results display properly
- ✓ Tables are scrollable horizontally

### TEST-027: Navigation Flow
**Objective:** Verify user can navigate through the feature

**Steps:**
1. Start at home page
2. Click "Import Data" nav link
3. Complete import
4. Click "Query Your Data" button
5. Return to import page

**Expected Results:**
- ✓ Each navigation step works correctly
- ✓ Active nav item is highlighted
- ✓ Browser back button works
- ✓ No broken links

---

## Test Suite 9: Data Persistence

### TEST-028: Imported Table Survives Server Restart
**Objective:** Verify imported data persists

**Steps:**
1. Import test_grades.csv as "student_grades"
2. Stop the server
3. Restart the server
4. Navigate to /agent
5. Query: "SELECT * FROM student_grades"

**Expected Results:**
- ✓ Table still exists after restart
- ✓ All data is intact
- ✓ Row count matches
- ✓ Comments are preserved

### TEST-029: Multiple Imports Persist
**Objective:** Verify multiple imported tables coexist

**Steps:**
1. Import test_grades.csv as "student_grades"
2. Import test_teachers.csv as "teacher_roster"
3. Restart server
4. Query schema to list all tables

**Expected Results:**
- ✓ Both tables exist
- ✓ Original school tables still exist
- ✓ All tables are queryable
- ✓ No data corruption

---

## Test Suite 10: Security Tests

### TEST-030: SQL Injection in Table Name
**Objective:** Verify protection against SQL injection

**Steps:**
1. Attempt to import with table name: "test'; DROP TABLE directory; --"
2. Observe behavior

**Expected Results:**
- ✓ Import fails with validation error
- ✓ No SQL is executed
- ✓ Directory table remains intact

### TEST-031: SQL Injection in Description
**Objective:** Verify description field is properly escaped

**Steps:**
1. Import with description containing SQL: "'; DELETE FROM directory WHERE '1'='1"
2. Complete import
3. Check database

**Expected Results:**
- ✓ Import succeeds
- ✓ Description is stored as literal text
- ✓ No SQL commands are executed
- ✓ All original tables remain

### TEST-032: Malicious CSV Content
**Objective:** Verify CSV content is sanitized

**Steps:**
1. Create CSV with formula injection: `=1+1, =cmd|'/c calc'`
2. Import the CSV
3. Query the data

**Expected Results:**
- ✓ Formulas are stored as text
- ✓ No code execution occurs
- ✓ Data is safely retrievable

---

## Test Suite 11: Integration with Existing Features

### TEST-033: Search Page Not Affected
**Objective:** Verify import doesn't break school search

**Steps:**
1. Import a custom table
2. Navigate to home page
3. Search for "Lincoln High"

**Expected Results:**
- ✓ Search works normally
- ✓ Only school data is shown
- ✓ No custom tables appear in results

### TEST-034: School Detail Pages Work
**Objective:** Verify school detail functionality intact

**Steps:**
1. Import custom data
2. Search for a school
3. Click to view details

**Expected Results:**
- ✓ Detail page loads
- ✓ AI scraper still works
- ✓ NAEP data still loads
- ✓ No errors in console

---

## Test Suite 12: Documentation and Help

### TEST-035: Help Text Clarity
**Objective:** Verify help text is useful for users

**Steps:**
1. Navigate to import page
2. Read all help text
3. Follow instructions

**Expected Results:**
- ✓ Instructions are clear
- ✓ Field help explains requirements
- ✓ "How it works" section is accurate
- ✓ No confusing jargon

### TEST-036: Error Messages Are Helpful
**Objective:** Verify error messages guide users to resolution

**Steps:**
1. Trigger various errors (duplicate table, invalid CSV, etc.)
2. Read error messages

**Expected Results:**
- ✓ Each error explains what went wrong
- ✓ Errors suggest how to fix the problem
- ✓ Technical details don't overwhelm users
- ✓ "Try Again" button is available

---

## Regression Test Checklist

After any code changes to the BYOD feature, run these critical tests:

- [ ] TEST-007: Basic import workflow
- [ ] TEST-013: Schema discovery
- [ ] TEST-014: Query imported data
- [ ] TEST-017: Duplicate table handling
- [ ] TEST-028: Data persistence after restart
- [ ] TEST-030: SQL injection protection
- [ ] TEST-033: Existing features not affected

---

## Performance Benchmarks

Target performance metrics:

| Operation | Target Time | Maximum Time |
|-----------|-------------|--------------|
| Small import (<1MB, <1000 rows) | <5 seconds | 10 seconds |
| Medium import (<10MB, <10K rows) | <15 seconds | 30 seconds |
| Large import (<100MB, <100K rows) | <60 seconds | 120 seconds |
| Query imported table | <2 seconds | 5 seconds |
| Schema discovery | <1 second | 3 seconds |
| AI description generation | <3 seconds | 10 seconds |

---

## Browser Compatibility

Test on:
- ✓ Chrome/Edge (latest)
- ✓ Firefox (latest)
- ✓ Safari (latest)
- ✓ Mobile Safari (iOS)
- ✓ Chrome Mobile (Android)

---

## Accessibility Tests

- [ ] Form labels are properly associated
- [ ] All interactive elements are keyboard accessible
- [ ] Error messages are announced to screen readers
- [ ] Color contrast meets WCAG AA standards
- [ ] Focus indicators are visible
- [ ] File upload works with assistive technology

---

## Notes for Test Implementation

### Automated Testing Recommendations

1. **Use Playwright or Cypress** for E2E automation
2. **Test data setup**: Create fixtures for all CSV test files
3. **Database cleanup**: Reset database between tests
4. **Mock AI responses**: For faster, more reliable tests
5. **Screenshot on failure**: Capture UI state for debugging

### Manual Testing Checklist

For release testing, manually verify:
- Complete workflow from start to finish
- Error messages are user-friendly
- Performance is acceptable
- Documentation is accurate
- No console errors

---

## Future Enhancements to Test

When these features are added, create test specs for:
- [ ] Edit/update imported tables
- [ ] Delete imported tables
- [ ] Import from URLs
- [ ] Import Excel files
- [ ] Preview data before import
- [ ] Data transformation options
- [ ] Scheduled imports
- [ ] Import history/audit log
