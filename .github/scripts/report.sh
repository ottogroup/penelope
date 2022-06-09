TEST_OUTPUT_FILE=$1

passed=$(grep -c "\-\-\- PASS:" "$TEST_OUTPUT_FILE")
failed=$(grep -c "\-\-\- FAIL:" "$TEST_OUTPUT_FILE")
skipped=$(grep -c "\-\-\- SKIP:" "$TEST_OUTPUT_FILE")

echo "# Summary" > "$GITHUB_STEP_SUMMARY"

{
  echo "## Overview"
  echo "| ✅ PASSED      | 🚫 FAILED    | ⏭ SKIPPED    |"
  echo "| -----------: | ---------: | ---------:  |"
  echo "| $passed     | $failed   | $skipped   |"
} >> "$GITHUB_STEP_SUMMARY"

AddListSection () {
  SectionName=$1;
  SectionLines=$2;

  echo "## $SectionName" >> "$GITHUB_STEP_SUMMARY"
  while read -r line ; do
    if [[ -z "$line" ]]; then
      echo "*none*" >> "$GITHUB_STEP_SUMMARY"
      break
    fi
    echo "* $line" >> "$GITHUB_STEP_SUMMARY"
  done <<< "$SectionLines"
}

AddListSection "FAILED" "$( grep "\-\-\- FAIL:" "$TEST_OUTPUT_FILE"  | cut -d " " -f 3)"
AddListSection "SKIPPED" "$(grep  "\-\-\- SKIP:" "$TEST_OUTPUT_FILE" | cut -d " " -f 3)"




