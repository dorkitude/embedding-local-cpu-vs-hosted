# A running/error settled phase already completed the initial embedding lane.
# Allow Requery to resume it; never start against an unfinished initial lane.
length == 16 and all(.[]; .status == "passed" or .evaluation_phase == "settled-v1")
