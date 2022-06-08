export POSTGRES_DB=backupdatabase
export POSTGRES_USER=backupuser
export POSTGRES_PASSWORD=backupuserpassword
export POSTGRES_PORT=5432

go test -v -p 1 ./... | tee tmp-test-output.txt

passed=$(cat tmp-test-output.txt | grep -c "\-\-\- PASS:")
failed=$(cat tmp-test-output.txt | grep -c "\-\-\- FAIL:")
skipped=$(cat tmp-test-output.txt | grep -c "\-\-\- SKIP:")

echo "# Summary" > $GITHUB_STEP_SUMMARY


echo "## Overview" > $GITHUB_STEP_SUMMARY
echo "| ✅ PASSED      | 🚫 FAILED    | ⏭ SKIPPED    |" >> $GITHUB_STEP_SUMMARY
echo "| -----------: | ---------: | ---------:  |" >> $GITHUB_STEP_SUMMARY
echo "| $passed     | $failed   | $skipped   |" >> $GITHUB_STEP_SUMMARY

echo "## FAILED" >> $GITHUB_STEP_SUMMARY
failed=$(cat tmp-test-output.txt | grep "\-\-\- FAILED:")
while read -r line ; do
    echo "* $(echo "$line" | cut -d " " -f 3)" >> $GITHUB_STEP_SUMMARY
done <<< "$failed"

echo "## SKIPPED" >> $GITHUB_STEP_SUMMARY
skipped=$(cat tmp-test-output.txt | grep "\-\-\- SKIP:")
while read -r line ; do
    echo "* $(echo "$line" | cut -d " " -f 3)" >> $GITHUB_STEP_SUMMARY
done <<< "$skipped"




