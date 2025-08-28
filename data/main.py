import csv
import os
import yaml
import analyze

base_dir = 'reports'

out = open('results.csv', 'a')

out.write(analyze.AnalysisUnit.header_row() + '\n')

out.flush()

index = 0

reports_count = len(os.listdir(base_dir))

print("Processing", reports_count, "reports...")

for report in os.listdir(base_dir):
    filename = os.fsdecode(report)
    if not filename.endswith(".yml"):
        continue

    print(filename)

    path = os.path.join(base_dir, report)
    if not os.path.isfile(path):
        continue

    query = ''

    try:
        with open(path, 'r') as stream:
            data = yaml.safe_load(stream)
            try:
                query = data["query"][-1]
            except KeyError as exc:
                print("Keyerror:", exc, "in", filename)
                continue
    except yaml.YAMLError as exc:
        print(exc)
        continue

    analyzer = analyze.AnalysisUnit(query)

    row = analyzer.to_row()
    out.write(f"{data["target"]}," + row + f",{data["strategy"] if "strategy" in data else 0}" + '\n')

out.close()
