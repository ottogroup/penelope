export POSTGRES_DB=backupdatabase
export POSTGRES_USER=backupuser
export POSTGRES_PASSWORD=backupuserpassword
export POSTGRES_PORT=5432

go test -v -p 1 ./... | tee tmp-test-output.txt

passed=$(cat tmp-test-output.txt | grep -c "\-\-\- PASS:")
failed=$(cat tmp-test-output.txt | grep -c "\-\-\- FAIL:")
skipped=$(cat tmp-test-output.txt | grep -c "\-\-\- SKIP:")

GITHUB_STEP_SUMMARY="Summary.md"

echo "# Summary" > $GITHUB_STEP_SUMMARY


echo "## Overview" > $GITHUB_STEP_SUMMARY
echo "| ✅ PASSED      | 🚫 FAILED    | ⏭ SKIPPED    |" >> $GITHUB_STEP_SUMMARY
echo "| -----------: | ---------: | ---------:  |" >> $GITHUB_STEP_SUMMARY
echo "| $passed     | $failed   | $skipped   |" >> $GITHUB_STEP_SUMMARY

AddListSection () {
  SectionName=$1;
  SectionLines=$2;

  echo "## $SectionName" >> $GITHUB_STEP_SUMMARY
  while read -r line ; do
    if [[ -z "$line" ]]; then
      echo "*none*" >> $GITHUB_STEP_SUMMARY
      break
    fi
    echo "* $line" >> $GITHUB_STEP_SUMMARY
  done <<< "$SectionLines"
}

AddListSection "FAILED" "$(cat tmp-test-output.txt | grep "\-\-\- FAIL:" | cut -d " " -f 3)"
AddListSection "SKIPPED" "$(cat tmp-test-output.txt | grep "\-\-\- SKIP:" | cut -d " " -f 3)"




