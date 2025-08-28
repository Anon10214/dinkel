#!/bin/bash

# Default to orig if no arguments passed
if [[ $# == 0 ]]; then
	set "orig"
fi

if [[ $1 != "orig" && $1 != "dinkel" && $1 != "dinkel-qc" && $1 != "dinkel-gs" && $1 != "dinkel-qc-gs" ]]; then
	echo "$1 is not a valid argument"
	printf "\nMust be one of:\n\torig\n\tdinkel\n\tdinkel-qc\n\tdinkel-gs\n\tdinkel-qc-gs\n\n"
	echo "orig being the runtime testing comparing different GDBMSs, whereas the others are the sensitivity analyses."
	exit 1
fi

docker run --rm -d --name dinkel-analysis-$1 \
	-v ./$1:/prometheus/data/snapshots/data \
	-p 9090:9090 \
	--entrypoint='/bin/sh' \
	prom/prometheus \
	-c '\
	cp -r /prometheus/data/snapshots/data/* /prometheus && \
	\
	prometheus \
	    --config.file=/etc/prometheus/prometheus.yml \
	    --storage.tsdb.path=/prometheus \
	'
if [[ $? != 0 ]]; then
	echo
	echo "Failed to start container, exiting."
	echo "Make sure no other containers are running from previous analyses and that port 9090 is free on your system."
	exit 1
fi

echo

printf "Container started, run\n\tdocker kill dinkel-analysis-$1\nto stop it.\n\n"

echo "Opening the URL for analysing the data"

url=""
case $1 in

	"orig")
		url="http://localhost:9090/graph?g0.expr=max_over_time(increase(dinkel_query_count[1d%3A])[1d%3A])&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=1d&g0.end_input=2023-12-04 09%3A27%3A00&g0.moment_input=2023-12-04 09%3A27%3A00&g0.step_input=8&g1.expr=max_over_time(increase(dinkel_total_keyword_count[1d%3A])[1d%3A])&g1.tab=0&g1.stacked=0&g1.show_exemplars=0&g1.range_input=1d&g1.end_input=2023-12-04 09%3A27%3A00&g1.moment_input=2023-12-04 09%3A27%3A00&g1.step_input=8&g2.expr=max_over_time(increase(dinkel_abstract_graph_summary_dependencies[1d%3A])[1d%3A])&g2.tab=0&g2.stacked=0&g2.show_exemplars=0&g2.range_input=1d&g2.end_input=2023-12-04 09%3A27%3A00&g2.moment_input=2023-12-04 09%3A27%3A00&g2.step_input=8&g3.expr=max_over_time(increase(dinkel_query_context_dependencies[1d%3A])[1d%3A])&g3.tab=0&g3.stacked=0&g3.show_exemplars=0&g3.range_input=1d&g3.end_input=2023-12-04 09%3A27%3A00&g3.moment_input=2023-12-04 09%3A27%3A00&g3.step_input=8&g4.expr=increase(dinkel_valid_query_count[1d])%0A%2F%0A(increase(dinkel_invalid_query_count[1d]) %2B increase(dinkel_valid_query_count[1d]))&g4.tab=0&g4.stacked=0&g4.show_exemplars=0&g4.range_input=1d&g4.end_input=2023-12-04 09%3A27%3A00&g4.moment_input=2023-12-04 09%3A27%3A00&g4.step_input=8&g5.expr=round((24 * 60) - max_over_time((max_over_time(timestamp(dinkel_query_count)[1d%3A]) - min_over_time(timestamp(dinkel_query_count)[1d%3A]))[1d%3A]) %2F 60)&g5.tab=0&g5.stacked=0&g5.show_exemplars=0&g5.range_input=30m&g5.end_input=2023-12-04 09%3A29%3A00&g5.moment_input=2023-12-04 09%3A29%3A00&g5.step_input=1"
		;;

	"dinkel")
		url="http://localhost:9090/graph?g0.expr=max_over_time(increase(dinkel_query_count[1d%3A])[1d%3A])&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=1d&g0.end_input=2024-01-04 13%3A32%3A00&g0.moment_input=2024-01-04 13%3A32%3A00&g1.expr=max_over_time(increase(dinkel_total_keyword_count[1d%3A])[1d%3A])&g1.tab=0&g1.stacked=0&g1.show_exemplars=0&g1.range_input=1d&g1.end_input=2024-01-04 13%3A32%3A00&g1.moment_input=2024-01-04 13%3A32%3A00&g1.step_input=8&g2.expr=max_over_time(increase(dinkel_abstract_graph_summary_dependencies[1d%3A])[1d%3A])&g2.tab=0&g2.stacked=0&g2.show_exemplars=0&g2.range_input=1d&g2.end_input=2024-01-04 13%3A32%3A00&g2.moment_input=2024-01-04 13%3A32%3A00&g2.step_input=8&g3.expr=max_over_time(increase(dinkel_query_context_dependencies[1d%3A])[1d%3A])&g3.tab=0&g3.stacked=0&g3.show_exemplars=0&g3.range_input=1d&g3.end_input=2024-01-04 13%3A32%3A00&g3.moment_input=2024-01-04 13%3A32%3A00&g3.step_input=8&g4.expr=increase(dinkel_valid_query_count[1d])%0A%2F%0A(increase(dinkel_invalid_query_count[1d]) %2B increase(dinkel_valid_query_count[1d]))&g4.tab=0&g4.stacked=0&g4.show_exemplars=0&g4.range_input=1d&g4.end_input=2024-01-04 13%3A32%3A00&g4.moment_input=2024-01-04 13%3A32%3A00&g4.step_input=8&g5.expr=round((24 * 60) - max_over_time((max_over_time(timestamp(dinkel_query_count)[1d%3A]) - min_over_time(timestamp(dinkel_query_count)[1d%3A]))[1d%3A]) %2F 60)&g5.tab=0&g5.stacked=0&g5.show_exemplars=0&g5.range_input=30m&g5.end_input=2024-01-04 13%3A34%3A00&g5.moment_input=2024-01-04 13%3A34%3A00&g5.step_input=1"
		;;

	"dinkel-qc")
		url="http://localhost:9090/graph?g0.expr=max_over_time(increase(dinkel_query_count[1d%3A])[1d%3A])&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=1d&g0.end_input=2024-01-05 13%3A41%3A00&g0.moment_input=2024-01-05 13%3A41%3A00&g0.step_input=8&g1.expr=max_over_time(increase(dinkel_total_keyword_count[1d%3A])[1d%3A])&g1.tab=0&g1.stacked=0&g1.show_exemplars=0&g1.range_input=1d&g1.end_input=2024-01-05 13%3A41%3A00&g1.moment_input=2024-01-05 13%3A41%3A00&g1.step_input=8&g2.expr=max_over_time(increase(dinkel_abstract_graph_summary_dependencies[1d%3A])[1d%3A])&g2.tab=0&g2.stacked=0&g2.show_exemplars=0&g2.range_input=1d&g2.end_input=2024-01-05 13%3A41%3A00&g2.moment_input=2024-01-05 13%3A41%3A00&g2.step_input=8&g3.expr=max_over_time(increase(dinkel_query_context_dependencies[1d%3A])[1d%3A])&g3.tab=0&g3.stacked=0&g3.show_exemplars=0&g3.range_input=1d&g3.end_input=2024-01-05 13%3A41%3A00&g3.moment_input=2024-01-05 13%3A41%3A00&g3.step_input=8&g4.expr=increase(dinkel_valid_query_count[1d])%0A%2F%0A(increase(dinkel_invalid_query_count[1d]) %2B increase(dinkel_valid_query_count[1d]))&g4.tab=0&g4.stacked=0&g4.show_exemplars=0&g4.range_input=1d&g4.end_input=2024-01-05 13%3A41%3A00&g4.moment_input=2024-01-05 13%3A41%3A00&g4.step_input=8&g5.expr=round((24 * 60) - max_over_time((max_over_time(timestamp(dinkel_query_count)[1d%3A]) - min_over_time(timestamp(dinkel_query_count)[1d%3A]))[1d%3A]) %2F 60)&g5.tab=0&g5.stacked=0&g5.show_exemplars=0&g5.range_input=30m&g5.end_input=2024-01-05 13%3A43%3A00&g5.moment_input=2024-01-05 13%3A43%3A00&g5.step_input=1"
		;;

	"dinkel-gs")
		url="http://localhost:9090/graph?g0.expr=max_over_time(increase(dinkel_query_count[1d%3A])[1d%3A])&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=1d&g0.end_input=2024-01-06 13%3A28%3A00&g0.moment_input=2024-01-06 13%3A28%3A00&g1.expr=max_over_time(increase(dinkel_total_keyword_count[1d%3A])[1d%3A])&g1.tab=0&g1.stacked=0&g1.show_exemplars=0&g1.range_input=1d&g1.end_input=2024-01-06 13%3A28%3A00&g1.moment_input=2024-01-06 13%3A28%3A00&g1.step_input=8&g2.expr=max_over_time(increase(dinkel_abstract_graph_summary_dependencies[1d%3A])[1d%3A])&g2.tab=0&g2.stacked=0&g2.show_exemplars=0&g2.range_input=1d&g2.end_input=2024-01-06 13%3A28%3A00&g2.moment_input=2024-01-06 13%3A28%3A00&g2.step_input=8&g3.expr=max_over_time(increase(dinkel_query_context_dependencies[1d%3A])[1d%3A])&g3.tab=0&g3.stacked=0&g3.show_exemplars=0&g3.range_input=1d&g3.end_input=2024-01-06 13%3A28%3A00&g3.moment_input=2024-01-06 13%3A28%3A00&g3.step_input=8&g4.expr=increase(dinkel_valid_query_count[1d])%0A%2F%0A(increase(dinkel_invalid_query_count[1d]) %2B increase(dinkel_valid_query_count[1d]))&g4.tab=0&g4.stacked=0&g4.show_exemplars=0&g4.range_input=1d&g4.end_input=2024-01-06 13%3A28%3A00&g4.moment_input=2024-01-06 13%3A28%3A00&g4.step_input=8&g5.expr=round((24 * 60) - max_over_time((max_over_time(timestamp(dinkel_query_count)[1d%3A]) - min_over_time(timestamp(dinkel_query_count)[1d%3A]))[1d%3A]) %2F 60)&g5.tab=0&g5.stacked=0&g5.show_exemplars=0&g5.range_input=30m&g5.end_input=2024-01-06 13%3A28%3A00&g5.moment_input=2024-01-06 13%3A28%3A00&g5.step_input=1"
		;;

	"dinkel-qc-gs")
		url="http://localhost:9090/graph?g0.expr=max_over_time(increase(dinkel_query_count[1d%3A])[1d%3A])&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=1d&g0.end_input=2024-01-06 13%3A26%3A00&g0.moment_input=2024-01-06 13%3A26%3A00&g1.expr=max_over_time(increase(dinkel_total_keyword_count[1d%3A])[1d%3A])&g1.tab=0&g1.stacked=0&g1.show_exemplars=0&g1.range_input=1d&g1.end_input=2024-01-06 13%3A26%3A00&g1.moment_input=2024-01-06 13%3A26%3A00&g1.step_input=8&g2.expr=max_over_time(increase(dinkel_abstract_graph_summary_dependencies[1d%3A])[1d%3A])&g2.tab=0&g2.stacked=0&g2.show_exemplars=0&g2.range_input=1d&g2.end_input=2024-01-06 13%3A26%3A00&g2.moment_input=2024-01-06 13%3A26%3A00&g2.step_input=8&g3.expr=max_over_time(increase(dinkel_query_context_dependencies[1d%3A])[1d%3A])&g3.tab=0&g3.stacked=0&g3.show_exemplars=0&g3.range_input=1d&g3.end_input=2024-01-06 13%3A26%3A00&g3.moment_input=2024-01-06 13%3A26%3A00&g3.step_input=8&g4.expr=increase(dinkel_valid_query_count[1d])%0A%2F%0A(increase(dinkel_invalid_query_count[1d]) %2B increase(dinkel_valid_query_count[1d]))&g4.tab=0&g4.stacked=0&g4.show_exemplars=0&g4.range_input=1d&g4.end_input=2024-01-06 13%3A26%3A00&g4.moment_input=2024-01-06 13%3A26%3A00&g4.step_input=8&g5.expr=round((24 * 60) - max_over_time((max_over_time(timestamp(dinkel_query_count)[1d%3A]) - min_over_time(timestamp(dinkel_query_count)[1d%3A]))[1d%3A]) %2F 60)&g5.tab=0&g5.stacked=0&g5.show_exemplars=0&g5.range_input=30m&g5.end_input=2024-01-06 13%3A28%3A00&g5.moment_input=2024-01-06 13%3A28%3A00&g5.step_input=1"
		;;
esac

xdg-open "$url" 2>/dev/null || open "$url" 2>/dev/null || printf "If your browser didn't open, navigate to the following URL to analyse the data:\n$url\n"

printf "\nThe panels show the following metrics:\n\t- total queries\n\t- total keywords\n\t- total graph summary dependencies\n\t- total query context dependencies\n\t- query validity rate\n\t- time remaining of analysis run\n"