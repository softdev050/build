// Copyright 2022 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

function Range(low, center, high, min, max, width, height, unit) {
	const margin = 40;
	const svg = d3.create("svg")
		.attr("width", width)
		.attr("height", height)
		.attr("viewBox", [0, 0, width, height])
		.attr("style", "max-width: 100%; height: auto; height: intrinsic;");

	const goodColor = "#005AB5";
	const badColor = "#DC3220";
	const pickColor = function(n) {
		const higherIsBetter = {
			"B/s": true,
			"ops/s": true
		};
		if (unit in higherIsBetter) {
			if (n > 0) {
				return goodColor;
			}
			return badColor;
		}
		if (n < 0) {
			return goodColor;
		}
		return badColor;
	};

	const xScale = d3.scaleLinear([min, max], [margin, width-margin]);
	const yBaseline = 3*height/4;

	// Draw line.
	const line = d3.line()
		.x(d => xScale(d))
		.y(yBaseline)

	const partialStroke = function() {
		return svg.append("path")
			.attr("fill", "none")
			.attr("stroke-width", 6);
	}
	if (high < 0) {
		partialStroke().attr("stroke", pickColor(high))
			.attr("d", line([low, high]));
	} else if (low < 0 && high > 0) {
		partialStroke().attr("stroke", pickColor(low))
			.attr("d", line([low, 0]));
		partialStroke().attr("stroke", pickColor(high))
			.attr("d", line([0, high]));
	} else {
		partialStroke().attr("stroke", pickColor(low))
			.attr("d", line([low, high]));
	}

	const tick = d3.line()
		.x(d => xScale(d[0]))
		.y(d => d[1])

	const xTicks = [low, center, high];
	for (const i in xTicks) {
		svg.append("path")
			.attr("fill", "none")
			.attr("stroke", pickColor(xTicks[i]))
			.attr("stroke-width", 2.5)
			.attr("d", tick([[xTicks[i], yBaseline-4], [xTicks[i], yBaseline+4]]))
	}

	svg.append("text")
		.attr("x", xScale(low)-4)
		.attr("y", yBaseline+3)
		.attr("fill", pickColor(low))
		.attr("text-anchor", "end")
		.attr("font-size", "11px")
		.text(Intl.NumberFormat([], {
			style: 'percent',
			signDisplay: 'always',
			minimumFractionDigits: 2,
		}).format(low));

	svg.append("text")
		.attr("x", xScale(center))
		.attr("y", height/2)
		.attr("fill", pickColor(center))
		.attr("text-anchor", "middle")
		.attr("font-size", "16px")
		.text(Intl.NumberFormat([], {
			style: 'percent',
			signDisplay: 'always',
			minimumFractionDigits: 2,
		}).format(center));

	svg.append("text")
		.attr("x", xScale(high)+4)
		.attr("y", yBaseline+3)
		.attr("fill", pickColor(high))
		.attr("text-anchor", "start")
		.attr("font-size", "11px")
		.text(Intl.NumberFormat([], {
			style: 'percent',
			signDisplay: 'always',
			minimumFractionDigits: 2,
		}).format(high));

	return svg.node();
}
