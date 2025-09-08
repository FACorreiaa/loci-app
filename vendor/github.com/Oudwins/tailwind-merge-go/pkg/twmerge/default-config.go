package twmerge

func getBreaks(groupId string) map[string]ClassPart {
	return map[string]ClassPart{
		"auto": {
			NextPart:     map[string]ClassPart{},
			Validators:   []ClassGroupValidator{},
			ClassGroupId: groupId,
		},
		"avoid": {
			NextPart:     make(map[string]ClassPart),
			Validators:   []ClassGroupValidator{},
			ClassGroupId: groupId,
		},
		"all": {
			NextPart:     map[string]ClassPart{},
			Validators:   []ClassGroupValidator{},
			ClassGroupId: groupId,
		},
		"page": {
			NextPart:     map[string]ClassPart{},
			Validators:   []ClassGroupValidator{},
			ClassGroupId: groupId,
		},
		"left": {
			NextPart:     map[string]ClassPart{},
			Validators:   []ClassGroupValidator{},
			ClassGroupId: groupId,
		},
		"right": {
			NextPart:     map[string]ClassPart{},
			Validators:   []ClassGroupValidator{},
			ClassGroupId: groupId,
		},
		"column": {
			NextPart:     map[string]ClassPart{},
			Validators:   []ClassGroupValidator{},
			ClassGroupId: groupId,
		},
	}
}

// This is horrible code. I'm sorry. I wanted to get the package working without writing the code to generate the config. Now that it is working I plan to writing it.
func MakeDefaultConfig() *TwMergeConfig {
	return &TwMergeConfig{
		ModifierSeparator: ':',
		ClassSeparator:    '-',
		ImportantModifier: '!',
		PostfixModifier:   '/',
		MaxCacheSize:      1000,
		// Prefix:            "",
		// theme:             TwTheme{},
		ConflictingClassGroups: ConflictingClassGroups{
			"overflow":         {"overflow-x", "overflow-y"},
			"overscroll":       {"overscroll-x", "overscroll-y"},
			"inset":            {"inset-x", "inset-y", "start", "end", "top", "right", "bottom", "left"},
			"inset-x":          {"right", "left"},
			"inset-y":          {"top", "bottom"},
			"flex":             {"basis", "grow", "shrink"},
			"gap":              {"gap-x", "gap-y"},
			"p":                {"px", "py", "ps", "pe", "pt", "pr", "pb", "pl"},
			"px":               {"pr", "pl"},
			"py":               {"pt", "pb"},
			"m":                {"mx", "my", "ms", "me", "mt", "mr", "mb", "ml"},
			"mx":               {"mr", "ml"},
			"my":               {"mt", "mb"},
			"size":             {"w", "h"},
			"font-size":        {"leading"},
			"fvn-normal":       {"fvn-ordinal", "fvn-slashed-zero", "fvn-figure", "fvn-spacing", "fvn-fraction"},
			"fvn-ordinal":      {"fvn-normal"},
			"fvn-slashed-zero": {"fvn-normal"},
			"fvn-figure":       {"fvn-normal"},
			"fvn-spacing":      {"fvn-normal"},
			"fvn-fraction":     {"fvn-normal"},
			"line-clamp":       {"display", "overflow"},
			"rounded":          {"rounded-s", "rounded-e", "rounded-t", "rounded-r", "rounded-b", "rounded-l", "rounded-ss", "rounded-se", "rounded-ee", "rounded-es", "rounded-tl", "rounded-tr", "rounded-br", "rounded-bl"},
			"rounded-s":        {"rounded-ss", "rounded-es"},
			"rounded-e":        {"rounded-se", "rounded-ee"},
			"rounded-t":        {"rounded-tl", "rounded-tr"},
			"rounded-r":        {"rounded-tr", "rounded-br"},
			"rounded-b":        {"rounded-br", "rounded-bl"},
			"rounded-l":        {"rounded-tl", "rounded-bl"},
			"border-spacing":   {"border-spacing-x", "border-spacing-y"},
			"border-w":         {"border-w-s", "border-w-e", "border-w-t", "border-w-r", "border-w-b", "border-w-l"},
			"border-w-x":       {"border-w-r", "border-w-l"},
			"border-w-y":       {"border-w-t", "border-w-b"},
			"border-color":     {"border-color-t", "border-color-r", "border-color-b", "border-color-l"},
			"border-color-x":   {"border-color-r", "border-color-l"},
			"border-color-y":   {"border-color-t", "border-color-b"},
			"scroll-m":         {"scroll-mx", "scroll-my", "scroll-ms", "scroll-me", "scroll-mt", "scroll-mr", "scroll-mb", "scroll-ml"},
			"scroll-mx":        {"scroll-mr", "scroll-ml"},
			"scroll-my":        {"scroll-mt", "scroll-mb"},
			"scroll-p":         {"scroll-px", "scroll-py", "scroll-ps", "scroll-pe", "scroll-pt", "scroll-pr", "scroll-pb", "scroll-pl"},
			"scroll-px":        {"scroll-pr", "scroll-pl"},
			"scroll-py":        {"scroll-pt", "scroll-pb"},
			"touch":            {"touch-x", "touch-y", "touch-pz"},
			"touch-x":          {"touch"},
			"touch-y":          {"touch"},
			"touch-pz":         {"touch"},
		},
		ClassGroups: ClassPart{
			NextPart: map[string]ClassPart{
				/**
				 * Aspect Ratio
				 * @see https://tailwindcss.com/docs/aspect-ratio
				 */
				"aspect": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "aspect",
						},
						"square": {
							ClassGroupId: "aspect",
						},
						"video": {
							ClassGroupId: "aspect",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "aspect",
						},
					},
				},
				/**
				 * Container
				 * @see https://tailwindcss.com/docs/container
				 */
				"container": {
					NextPart:     map[string]ClassPart{},
					ClassGroupId: "container",
				},

				/**
				 * Columns
				 * @see https://tailwindcss.com/docs/columns
				 */
				"columns": {
					NextPart: map[string]ClassPart{},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsTshirtSize,
							ClassGroupId: "columns",
						},
					},
				},

				"break": {
					NextPart: map[string]ClassPart{

						/**
						 * Break After
						 * @see https://tailwindcss.com/docs/break-after
						 */
						"after": {
							NextPart: getBreaks("break-after"),
						},

						/** Break Before @see https://tailwindcss.com/docs/break-before
						 */
						"before": {
							NextPart: getBreaks("break-before"),
						},

						/**
						 * Break Inside
						 * @see https://tailwindcss.com/docs/break-inside
						 */
						"inside": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "break-inside",
								},
								"avoid": {
									NextPart: map[string]ClassPart{
										"page": {
											ClassGroupId: "break-inside",
										},
										"column": {
											ClassGroupId: "break-inside",
										},
									},
									ClassGroupId: "break-inside",
								},
							},
						},

						/**
						 * Word Break
						 * @see https://tailwindcss.com/docs/word-break
						 */

						"normal": {
							ClassGroupId: "break",
						},
						"words": {
							ClassGroupId: "break",
						},
						"all": {
							ClassGroupId: "break",
						},
						"keep": {
							ClassGroupId: "break",
						},
					},
					Validators: []ClassGroupValidator{},
				},

				"box": {
					NextPart: map[string]ClassPart{
						/**
						 * Box Sizing
						 * @see https://tailwindcss.com/docs/box-sizing
						 */

						"border": {
							ClassGroupId: "box",
						},
						"content": {
							ClassGroupId: "box",
						},

						/**
						 * Box Decoration Break
						 * @see https://tailwindcss.com/docs/box-decoration-break
						 */

						"decoration": {
							NextPart: map[string]ClassPart{
								"slice": {
									ClassGroupId: "box-decoration"},
								"clone": {
									ClassGroupId: "box-decoration",
								},
							},
						},
					},
				},

				/**
				 * Display
				 * @see https://tailwindcss.com/docs/display
				 */

				"block": {
					ClassGroupId: "display",
				},
				"inline": {
					NextPart: map[string]ClassPart{
						"block": {ClassGroupId: "display"},
						"flex":  {ClassGroupId: "display"},
						"grid":  {ClassGroupId: "display"},
						"table": {ClassGroupId: "display"},
					},
					ClassGroupId: "display",
				},
				"flex": {
					NextPart: map[string]ClassPart{
						"row": {
							NextPart: map[string]ClassPart{
								"reverse": {
									ClassGroupId: "flex-direction",
								},
							},
							ClassGroupId: "flex-direction",
						},
						"col": {
							NextPart: map[string]ClassPart{
								"reverse": {
									ClassGroupId: "flex-direction",
								},
							},
							ClassGroupId: "flex-direction",
						},
						"wrap": {
							NextPart: map[string]ClassPart{
								"reverse": {
									ClassGroupId: "flex-wrap",
								},
							},
							ClassGroupId: "flex-wrap",
						},
						"nowrap": {
							ClassGroupId: "flex-wrap",
						},
						"1": {
							ClassGroupId: "flex",
						},
						"auto": {
							ClassGroupId: "flex",
						},
						"initial": {
							ClassGroupId: "flex",
						},
						"none": {
							ClassGroupId: "flex",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "flex",
						},
					},
					ClassGroupId: "display",
				},
				"table": {
					NextPart: map[string]ClassPart{
						"caption": {
							ClassGroupId: "display",
						},
						"cell": {
							ClassGroupId: "display",
						},
						"column": {
							NextPart: map[string]ClassPart{
								"group": {
									ClassGroupId: "display",
								},
							},
							ClassGroupId: "display",
						},
						"footer": {
							NextPart: map[string]ClassPart{
								"group": {
									ClassGroupId: "display",
								},
							},
						},
						"header": {
							NextPart: map[string]ClassPart{
								"group": {
									ClassGroupId: "display",
								},
							},
						},
						"row": {
							NextPart: map[string]ClassPart{
								"group": {
									ClassGroupId: "display",
								},
							},
							ClassGroupId: "display",
						},
						"auto": {
							ClassGroupId: "table-layout",
						},
						"fixed": {
							ClassGroupId: "table-layout",
						},
					},
					ClassGroupId: "display",
				},
				"flow": {
					NextPart: map[string]ClassPart{"root": {ClassGroupId: "display"}},
				},
				"grid": {
					NextPart: map[string]ClassPart{
						"cols": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsAny,
									ClassGroupId: "grid-cols",
								},
							},
						},
						"rows": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsAny,
									ClassGroupId: "grid-rows",
								},
							},
						},
						"flow": {
							NextPart: map[string]ClassPart{
								"row": {
									NextPart: map[string]ClassPart{
										"dense": {
											ClassGroupId: "grid-flow",
										},
									},
									ClassGroupId: "grid-flow",
								},
								"col": {
									NextPart: map[string]ClassPart{
										"dense": {
											ClassGroupId: "grid-flow",
										},
									},
									ClassGroupId: "grid-flow",
								},
								"dense": {
									ClassGroupId: "grid-flow",
								},
							},
						},
					},
					Validators:   []ClassGroupValidator{},
					ClassGroupId: "display",
				},
				"contents": {ClassGroupId: "display"},
				"list": {
					NextPart: map[string]ClassPart{
						"item": {
							ClassGroupId: "display",
						},
						"image": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "list-image",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "list-image",
								},
							},
						},
						"none": {
							ClassGroupId: "list-style-type",
						},
						"disc": {
							ClassGroupId: "list-style-type",
						},
						"decimal": {
							ClassGroupId: "list-style-type",
						},
						"inside": {
							ClassGroupId: "list-style-position",
						},
						"outside": {
							ClassGroupId: "list-style-position",
						},
					},
					Validators: []ClassGroupValidator{
						{
							// fn : TODO: You need to provide the function implementation here
							ClassGroupId: "list-style-type",
						},
					},
				},
				"hidden": {ClassGroupId: "display"},
				"float": {
					NextPart: map[string]ClassPart{
						"right": {
							ClassGroupId: "float",
						},
						"left": {
							ClassGroupId: "float",
						},
						"none": {
							ClassGroupId: "float",
						},
						"start": {
							ClassGroupId: "float",
						},
						"end": {
							ClassGroupId: "float",
						},
					},
				},
				"clear": {
					NextPart: map[string]ClassPart{
						"left": {
							ClassGroupId: "clear",
						},
						"right": {
							ClassGroupId: "clear",
						},
						"both": {
							ClassGroupId: "clear",
						},
						"none": {
							ClassGroupId: "clear",
						},
						"start": {
							ClassGroupId: "clear",
						},
						"end": {
							ClassGroupId: "clear",
						},
					},
				},
				"isolate": {ClassGroupId: "isolation"},
				"isolation": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "isolation",
						},
					},
				},
				"object": {
					NextPart: map[string]ClassPart{
						"contain": {
							ClassGroupId: "object-fit",
						},
						"cover": {
							ClassGroupId: "object-fit",
						},
						"fill": {
							ClassGroupId: "object-fit",
						},
						"none": {
							ClassGroupId: "object-fit",
						},
						"scale": {
							NextPart: map[string]ClassPart{
								"down": {
									ClassGroupId: "object-fit",
								},
							},
						},
						"bottom": {
							ClassGroupId: "object-position",
						},
						"center": {
							ClassGroupId: "object-position",
						},
						"left": {
							NextPart: map[string]ClassPart{
								"bottom": {
									ClassGroupId: "object-position",
								},
								"top": {
									ClassGroupId: "object-position",
								},
							},
						},
						"right": {
							NextPart: map[string]ClassPart{
								"bottom": {
									ClassGroupId: "object-position",
								},
								"top": {
									ClassGroupId: "object-position",
								},
							},
						},
						"top": {
							ClassGroupId: "object-position",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "object-position",
						},
					},
				},

				"overflow": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "overflow",
						},
						"hidden": {
							ClassGroupId: "overflow",
						},
						"clip": {
							ClassGroupId: "overflow",
						},
						"visible": {
							ClassGroupId: "overflow",
						},
						"scroll": {
							ClassGroupId: "overflow",
						},
						"x": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "overflow-x",
								},
								"hidden": {
									ClassGroupId: "overflow-x",
								},
								"clip": {
									ClassGroupId: "overflow-x",
								},
								"visible": {
									ClassGroupId: "overflow-x",
								},
								"scroll": {
									ClassGroupId: "overflow-x",
								},
							},
						},
						"y": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "overflow-y",
								},
								"hidden": {
									ClassGroupId: "overflow-y",
								},
								"clip": {
									ClassGroupId: "overflow-y",
								},
								"visible": {
									ClassGroupId: "overflow-y",
								},
								"scroll": {
									ClassGroupId: "overflow-y",
								},
							},
						},
					},
				},
				"overscroll": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "overscroll",
						},
						"contain": {
							ClassGroupId: "overscroll",
						},
						"none": {
							ClassGroupId: "overscroll",
						},
						"x": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "overscroll-x",
								},
								"contain": {
									ClassGroupId: "overscroll-x",
								},
								"none": {
									ClassGroupId: "overscroll-x",
								},
							},
						},
						"y": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "overscroll-y",
								},
								"contain": {
									ClassGroupId: "overscroll-y",
								},
								"none": {
									ClassGroupId: "overscroll-y",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{},
				},

				"static": {
					ClassGroupId: "position",
				},
				"fixed": {
					ClassGroupId: "position",
				},
				"absolute": {
					ClassGroupId: "position",
				},
				"relative": {
					ClassGroupId: "position",
				},
				"sticky": {
					ClassGroupId: "position",
				},

				"inset": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "inset",
						},
						"x": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "inset-x",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "inset-x",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "inset-x",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "inset-x",
								},
							},
						},
						"y": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "inset-y",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "inset-y",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "inset-y",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "inset-y",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "inset",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "inset",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "inset",
						},
					},
				},
				"start": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "start",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "start",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "start",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "start",
						},
					},
				},
				"end": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "end",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "end",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "end",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "end",
						},
					},
				},
				"top": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "top",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "top",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "top",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "top",
						},
					},
				},
				"right": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "right",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "right",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "right",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "right",
						},
					},
				},
				"bottom": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "bottom",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "bottom",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "bottom",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "bottom",
						},
					},
				},
				"left": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "left",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "left",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "left",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "left",
						},
					},
				},
				"visible": {
					ClassGroupId: "visibility",
				},
				"invisible": {
					ClassGroupId: "visibility",
				},
				"collapse": {
					ClassGroupId: "visibility",
				},
				"z": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "z",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsInteger,
							ClassGroupId: "z",
						},
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "z",
						},
					},
				},
				"basis": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "basis",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "basis",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "basis",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "basis",
						},
					},
				},
				"grow": {
					NextPart: map[string]ClassPart{
						"0": {
							ClassGroupId: "grow",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "grow",
						},
					},
					ClassGroupId: "grow",
				},
				"shrink": {
					NextPart: map[string]ClassPart{
						"0": {
							ClassGroupId: "shrink",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "shrink",
						},
					},
					ClassGroupId: "shrink",
				},
				"order": {
					NextPart: map[string]ClassPart{
						"first": {
							ClassGroupId: "order",
						},
						"last": {
							ClassGroupId: "order",
						},
						"none": {
							ClassGroupId: "order",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsInteger,
							ClassGroupId: "order",
						},
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "order",
						},
					},
				},
				"col": {
					NextPart: map[string]ClassPart{
						"auto": {
							NextPart:     map[string]ClassPart{},
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "col-start-end",
						},
						"span": {
							NextPart: map[string]ClassPart{
								"full": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "col-start-end",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsInteger,
									ClassGroupId: "col-start-end",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "col-start-end",
								},
							},
						},
						"start": {
							NextPart: map[string]ClassPart{
								"auto": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "col-start",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "col-start",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "col-start",
								},
							},
						},
						"end": {
							NextPart: map[string]ClassPart{
								"auto": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "col-end",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "col-end",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "col-end",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "col-start-end",
						},
					},
				},
				"row": {
					NextPart: map[string]ClassPart{
						"auto": {
							NextPart:     map[string]ClassPart{},
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "row-start-end",
						},
						"span": {
							NextPart: map[string]ClassPart{},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsInteger,
									ClassGroupId: "row-start-end",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "row-start-end",
								},
							},
						},
						"start": {
							NextPart: map[string]ClassPart{
								"auto": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "row-start",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "row-start",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "row-start",
								},
							},
						},
						"end": {
							NextPart: map[string]ClassPart{
								"auto": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "row-end",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "row-end",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "row-end",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "row-start-end",
						},
					},
				},
				"auto": {
					NextPart: map[string]ClassPart{
						"cols": {
							NextPart: map[string]ClassPart{
								"auto": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "auto-cols",
								},
								"min": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "auto-cols",
								},
								"max": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "auto-cols",
								},
								"fr": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "auto-cols",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "auto-cols",
								},
							},
						},
						"rows": {
							NextPart: map[string]ClassPart{
								"auto": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "auto-rows",
								},
								"min": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "auto-rows",
								},
								"max": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "auto-rows",
								},
								"fr": {
									NextPart:     map[string]ClassPart{},
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "auto-rows",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "auto-rows",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{},
				},
				"gap": {
					NextPart: map[string]ClassPart{
						"x": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "gap-x",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "gap-x",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "gap-x",
								},
							},
						},
						"y": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "gap-y",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "gap-y",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "gap-y",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "gap",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "gap",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "gap",
						},
					},
				},
				"justify": {
					NextPart: map[string]ClassPart{
						"normal": {
							ClassGroupId: "justify-content",
						},
						"start": {
							ClassGroupId: "justify-content",
						},
						"end": {
							ClassGroupId: "justify-content",
						},
						"center": {
							ClassGroupId: "justify-content",
						},
						"between": {
							ClassGroupId: "justify-content",
						},
						"around": {
							ClassGroupId: "justify-content",
						},
						"evenly": {
							ClassGroupId: "justify-content",
						},
						"stretch": {
							ClassGroupId: "justify-content",
						},
						"items": {
							NextPart: map[string]ClassPart{
								"start": {
									ClassGroupId: "justify-items",
								},
								"end": {
									ClassGroupId: "justify-items",
								},
								"center": {
									ClassGroupId: "justify-items",
								},
								"stretch": {
									ClassGroupId: "justify-items",
								},
							},
						},
						"self": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "justify-self",
								},
								"start": {
									ClassGroupId: "justify-self",
								},
								"end": {
									ClassGroupId: "justify-self",
								},
								"center": {
									ClassGroupId: "justify-self",
								},
								"stretch": {
									ClassGroupId: "justify-self",
								},
							},
						},
					},
				},
				"content": {
					NextPart: map[string]ClassPart{
						"normal": {
							ClassGroupId: "align-content",
						},
						"start": {
							ClassGroupId: "align-content",
						},
						"end": {
							ClassGroupId: "align-content",
						},
						"center": {
							ClassGroupId: "align-content",
						},
						"between": {
							ClassGroupId: "align-content",
						},
						"around": {
							ClassGroupId: "align-content",
						},
						"evenly": {
							ClassGroupId: "align-content",
						},
						"stretch": {
							ClassGroupId: "align-content",
						},
						"baseline": {
							ClassGroupId: "align-content",
						},
						"none": {
							ClassGroupId: "content",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "content",
						},
					},
				},
				"items": {
					NextPart: map[string]ClassPart{
						"start": {
							ClassGroupId: "align-items",
						},
						"end": {
							ClassGroupId: "align-items",
						},
						"center": {
							ClassGroupId: "align-items",
						},
						"baseline": {
							ClassGroupId: "align-items",
						},
						"stretch": {
							ClassGroupId: "align-items",
						},
					},
				},
				"self": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "align-self",
						},
						"start": {
							ClassGroupId: "align-self",
						},
						"end": {
							ClassGroupId: "align-self",
						},
						"center": {
							ClassGroupId: "align-self",
						},
						"stretch": {
							ClassGroupId: "align-self",
						},
						"baseline": {
							ClassGroupId: "align-self",
						},
					},
				},
				"place": {
					NextPart: map[string]ClassPart{
						"content": {
							NextPart: map[string]ClassPart{
								"start": {
									ClassGroupId: "place-content",
								},
								"end": {
									ClassGroupId: "place-content",
								},
								"center": {
									ClassGroupId: "place-content",
								},
								"between": {
									ClassGroupId: "place-content",
								},
								"around": {
									ClassGroupId: "place-content",
								},
								"evenly": {
									ClassGroupId: "place-content",
								},
								"stretch": {
									ClassGroupId: "place-content",
								},
								"baseline": {
									ClassGroupId: "place-content",
								},
							},
						},
						"items": {
							NextPart: map[string]ClassPart{
								"start": {
									ClassGroupId: "place-items",
								},
								"end": {
									ClassGroupId: "place-items",
								},
								"center": {
									ClassGroupId: "place-items",
								},
								"baseline": {
									ClassGroupId: "place-items",
								},
								"stretch": {
									ClassGroupId: "place-items",
								},
							},
						},
						"self": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "place-self",
								},
								"start": {
									ClassGroupId: "place-self",
								},
								"end": {
									ClassGroupId: "place-self",
								},
								"center": {
									ClassGroupId: "place-self",
								},
								"stretch": {
									ClassGroupId: "place-self",
								},
							},
						},
					},
				},
				"p": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "p",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "p",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "p",
						},
					},
				},
				"px": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "px",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "px",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "px",
						},
					},
				},
				"py": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "py",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "py",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "py",
						},
					},
				},
				"ps": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "ps",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "ps",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "ps",
						},
					},
				},
				"pe": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "pe",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "pe",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "pe",
						},
					},
				},
				"pt": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "pt",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "pt",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "pt",
						},
					},
				},
				"pr": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "pr",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "pr",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "pr",
						},
					},
				},
				"pb": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "pb",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "pb",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "pb",
						},
					},
				},
				"pl": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "pl",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "pl",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "pl",
						},
					},
				},
				"m": {
					NextPart: map[string]ClassPart{
						"auto": {
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "m",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "m",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "m",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "m",
						},
					},
				},
				"mx": {
					NextPart: map[string]ClassPart{
						"auto": {
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "mx",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "mx",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "mx",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "mx",
						},
					},
				},
				"my": {
					NextPart: map[string]ClassPart{
						"auto": {
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "my",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "my",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "my",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "my",
						},
					},
				},
				"ms": {
					NextPart: map[string]ClassPart{
						"auto": {
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "ms",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "ms",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "ms",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "ms",
						},
					},
				},
				"me": {
					NextPart: map[string]ClassPart{
						"auto": {
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "me",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "me",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "me",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "me",
						},
					},
				},
				"mt": {
					NextPart: map[string]ClassPart{
						"auto": {
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "mt",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "mt",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "mt",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "mt",
						},
					},
				},
				"mr": {
					NextPart: map[string]ClassPart{
						"auto": {
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "mr",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "mr",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "mr",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "mr",
						},
					},
				},
				"mb": {
					NextPart: map[string]ClassPart{
						"auto": {
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "mb",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "mb",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "mb",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "mb",
						},
					},
				},
				"ml": {
					NextPart: map[string]ClassPart{
						"auto": {
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "ml",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "ml",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "ml",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "ml",
						},
					},
				},
				"space": {
					NextPart: map[string]ClassPart{
						"x": {
							NextPart: map[string]ClassPart{
								"reverse": {
									Validators:   []ClassGroupValidator{},
									ClassGroupId: "space-x-reverse",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "space-x",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "space-x",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "space-x",
								},
							},
						},
						"y": {
							NextPart: map[string]ClassPart{
								"reverse": {
									ClassGroupId: "space-y-reverse",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "space-y",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "space-y",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "space-y",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{},
				},
				"w": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "w",
						},
						"min": {
							ClassGroupId: "w",
						},
						"max": {
							ClassGroupId: "w",
						},
						"fit": {
							ClassGroupId: "w",
						},
						"svw": {
							ClassGroupId: "w",
						},
						"lvw": {
							ClassGroupId: "w",
						},
						"dvw": {
							ClassGroupId: "w",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "w",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "w",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "w",
						},
					},
				},
				"min": {
					NextPart: map[string]ClassPart{
						"w": {
							NextPart: map[string]ClassPart{
								"min": {
									ClassGroupId: "min-w",
								},
								"max": {
									ClassGroupId: "min-w",
								},
								"fit": {
									ClassGroupId: "min-w",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "min-w",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "min-w",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "min-w",
								},
							},
						},
						"h": {
							NextPart: map[string]ClassPart{
								"min": {
									ClassGroupId: "min-h",
								},
								"max": {
									ClassGroupId: "min-h",
								},
								"fit": {
									ClassGroupId: "min-h",
								},
								"svh": {
									ClassGroupId: "min-h",
								},
								"lvh": {
									ClassGroupId: "min-h",
								},
								"dvh": {
									ClassGroupId: "min-h",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "min-h",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "min-h",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "min-h",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{},
				},
				"max": {
					NextPart: map[string]ClassPart{
						"w": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "max-w",
								},
								"full": {
									ClassGroupId: "max-w",
								},
								"min": {
									ClassGroupId: "max-w",
								},
								"max": {
									ClassGroupId: "max-w",
								},
								"fit": {
									ClassGroupId: "max-w",
								},
								"prose": {
									ClassGroupId: "max-w",
								},
								"screen": {
									Validators: []ClassGroupValidator{
										{
											Fn:           IsTshirtSize,
											ClassGroupId: "max-w",
										},
									},
									ClassGroupId: "max-w",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "max-w",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "max-w",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "max-w",
								},
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "max-w",
								},
							},
							ClassGroupId: "max-w",
						},
						"h": {
							NextPart: map[string]ClassPart{
								"min": {
									ClassGroupId: "max-h",
								},
								"max": {
									ClassGroupId: "max-h",
								},
								"fit": {
									ClassGroupId: "max-h",
								},
								"svh": {
									ClassGroupId: "max-h",
								},
								"lvh": {
									ClassGroupId: "max-h",
								},
								"dvh": {
									ClassGroupId: "max-h",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "max-h",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "max-h",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "max-h",
								},
							},
							ClassGroupId: "max-h",
						},
					},
				},
				"h": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "h",
						},
						"min": {
							ClassGroupId: "h",
						},
						"max": {
							ClassGroupId: "h",
						},
						"fit": {
							ClassGroupId: "h",
						},
						"svh": {
							ClassGroupId: "h",
						},
						"lvh": {
							ClassGroupId: "h",
						},
						"dvh": {
							ClassGroupId: "h",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "h",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "h",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "h",
						},
					},
					ClassGroupId: "h",
				},
				"size": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "size",
						},
						"min": {
							ClassGroupId: "size",
						},
						"max": {
							ClassGroupId: "size",
						},
						"fit": {
							ClassGroupId: "size",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "size",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "size",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "size",
						},
					},
					ClassGroupId: "size",
				},
				"text": {
					NextPart: map[string]ClassPart{
						"base": {
							ClassGroupId: "font-size",
						},
						"left": {
							ClassGroupId: "text-alignment",
						},
						"center": {
							ClassGroupId: "text-alignment",
						},
						"right": {
							ClassGroupId: "text-alignment",
						},
						"justify": {
							ClassGroupId: "text-alignment",
						},
						"start": {
							ClassGroupId: "text-alignment",
						},
						"end": {
							ClassGroupId: "text-alignment",
						},
						"opacity": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "text-opacity",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "text-opacity",
								},
							},
							ClassGroupId: "text-opacity",
						},
						"ellipsis": {
							ClassGroupId: "text-overflow",
						},
						"clip": {
							ClassGroupId: "text-overflow",
						},
						"wrap": {
							ClassGroupId: "text-wrap",
						},
						"nowrap": {
							ClassGroupId: "text-wrap",
						},
						"balance": {
							ClassGroupId: "text-wrap",
						},
						"pretty": {
							ClassGroupId: "text-wrap",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsTshirtSize,
							ClassGroupId: "font-size",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "font-size",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "text-color",
						},
					},
				},
				"antialiased": {
					ClassGroupId: "font-smoothing",
				},
				"subpixel": {
					NextPart: map[string]ClassPart{
						"antialiased": {
							ClassGroupId: "font-smoothing",
						},
					},
				},
				"italic": {
					ClassGroupId: "font-style",
				},
				"not": {
					NextPart: map[string]ClassPart{
						"italic": {
							ClassGroupId: "font-style",
						},
						"sr": {
							NextPart: map[string]ClassPart{
								"only": {
									ClassGroupId: "sr",
								},
							},
						},
					},
				},
				"font": {
					NextPart: map[string]ClassPart{
						"thin": {
							ClassGroupId: "font-weight",
						},
						"extralight": {
							ClassGroupId: "font-weight",
						},
						"light": {
							ClassGroupId: "font-weight",
						},
						"normal": {
							ClassGroupId: "font-weight",
						},
						"medium": {
							ClassGroupId: "font-weight",
						},
						"semibold": {
							ClassGroupId: "font-weight",
						},
						"bold": {
							ClassGroupId: "font-weight",
						},
						"extrabold": {
							ClassGroupId: "font-weight",
						},
						"black": {
							ClassGroupId: "font-weight",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryNumber,
							ClassGroupId: "font-weight",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "font-family",
						},
					},
				},
				"normal": {
					NextPart: map[string]ClassPart{
						"nums": {
							ClassGroupId: "fvn-normal",
						},
						"case": {
							ClassGroupId: "text-transform",
						},
					},
				},
				"ordinal": {
					ClassGroupId: "fvn-ordinal",
				},
				"slashed": {
					NextPart: map[string]ClassPart{
						"zero": {
							ClassGroupId: "fvn-slashed-zero",
						},
					},
				},
				"lining": {
					NextPart: map[string]ClassPart{
						"nums": {
							ClassGroupId: "fvn-figure",
						},
					},
				},
				"oldstyle": {
					NextPart: map[string]ClassPart{
						"nums": {
							ClassGroupId: "fvn-figure",
						},
					},
				},
				"proportional": {
					NextPart: map[string]ClassPart{
						"nums": {
							ClassGroupId: "fvn-spacing",
						},
					},
				},
				"tabular": {
					NextPart: map[string]ClassPart{
						"nums": {
							ClassGroupId: "fvn-spacing",
						},
					},
				},
				"diagonal": {
					NextPart: map[string]ClassPart{
						"fractions": {
							ClassGroupId: "fvn-fraction",
						},
					},
				},
				"stacked": {
					NextPart: map[string]ClassPart{
						"fractons": {
							ClassGroupId: "fvn-fraction",
						},
					},
				},
				"tracking": {
					NextPart: map[string]ClassPart{
						"tighter": {
							ClassGroupId: "tracking",
						},
						"tight": {
							ClassGroupId: "tracking",
						},
						"normal": {
							ClassGroupId: "tracking",
						},
						"wide": {
							ClassGroupId: "tracking",
						},
						"wider": {
							ClassGroupId: "tracking",
						},
						"widest": {
							ClassGroupId: "tracking",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "tracking",
						},
					},
				},
				"line": {
					NextPart: map[string]ClassPart{
						"clamp": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "line-clamp",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "line-clamp",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "line-clamp",
								},
							},
						},
						"through": {
							ClassGroupId: "text-decoration",
						},
					},
				},
				"leading": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "leading",
						},
						"tight": {
							ClassGroupId: "leading",
						},
						"snug": {
							ClassGroupId: "leading",
						},
						"normal": {
							ClassGroupId: "leading",
						},
						"relaxed": {
							ClassGroupId: "leading",
						},
						"loose": {
							ClassGroupId: "leading",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsLength,
							ClassGroupId: "leading",
						},
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "leading",
						},
					},
				},
				"placeholder": {
					NextPart: map[string]ClassPart{
						"opacity": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "placeholder-opacity",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "placeholder-opacity",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsAny,
							ClassGroupId: "placeholder-color",
						},
					},
				},
				"underline": {
					NextPart: map[string]ClassPart{
						"offset": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "underline-offset",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "underline-offset",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "underline-offset",
								},
							},
						},
					},
					ClassGroupId: "text-decoration",
				},
				"overline": {
					ClassGroupId: "text-decoration",
				},
				"no": {
					NextPart: map[string]ClassPart{
						"underline": {
							ClassGroupId: "text-decoration",
						},
					},
				},
				"decoration": {
					NextPart: map[string]ClassPart{
						"solid": {
							ClassGroupId: "text-decoration-style",
						},
						"dashed": {
							ClassGroupId: "text-decoration-style",
						},
						"dotted": {
							ClassGroupId: "text-decoration-style",
						},
						"double": {
							ClassGroupId: "text-decoration-style",
						},
						"none": {
							ClassGroupId: "text-decoration-style",
						},
						"wavy": {
							ClassGroupId: "text-decoration-style",
						},
						"auto": {
							ClassGroupId: "text-decoration-thickness",
						},
						"from": {
							NextPart: map[string]ClassPart{
								"font": {
									ClassGroupId: "text-decoration-thickness",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsLength,
							ClassGroupId: "text-decoration-thickness",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "text-decoration-thickness",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "text-decoration-color",
						},
					},
					ClassGroupId: "",
				},
				"uppercase": {
					ClassGroupId: "text-transform",
				},
				"lowercase": {
					ClassGroupId: "text-transform",
				},
				"capitalize": {
					ClassGroupId: "text-transform",
				},
				"truncate": {
					ClassGroupId: "text-overflow",
				},
				"indent": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "indent",
						},
						{
							Fn:           IsLength,
							ClassGroupId: "indent",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "indent",
						},
					},
				},
				"align": {
					NextPart: map[string]ClassPart{
						"baseline": {
							ClassGroupId: "vertical-align",
						},
						"top": {
							ClassGroupId: "vertical-align",
						},
						"middle": {
							ClassGroupId: "vertical-align",
						},
						"bottom": {
							ClassGroupId: "vertical-align",
						},
						"text": {
							NextPart: map[string]ClassPart{
								"top": {
									ClassGroupId: "vertical-align",
								},
								"bottom": {
									ClassGroupId: "vertical-align",
								},
							},
						},
						"sub": {
							ClassGroupId: "vertical-align",
						},
						"super": {
							ClassGroupId: "vertical-align",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "vertical-align",
						},
					},
				},
				"whitespace": {
					NextPart: map[string]ClassPart{
						"normal": {
							ClassGroupId: "whitespace",
						},
						"nowrap": {
							ClassGroupId: "whitespace",
						},
						"pre": {
							NextPart: map[string]ClassPart{
								"line": {
									ClassGroupId: "whitespace",
								},
								"wrap": {
									ClassGroupId: "whitespace",
								},
							},
							ClassGroupId: "whitespace",
						},
						"break": {
							NextPart: map[string]ClassPart{
								"spaces": {
									ClassGroupId: "whitespace",
								},
							},
							ClassGroupId: "",
						},
					},
				},
				"hyphens": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "hyphens",
						},
						"manual": {
							ClassGroupId: "hyphens",
						},
						"auto": {
							ClassGroupId: "hyphens",
						},
					},
				},
				"bg": {
					NextPart: map[string]ClassPart{
						"fixed": {
							ClassGroupId: "bg-attachment",
						},
						"local": {
							ClassGroupId: "bg-attachment",
						},
						"scroll": {
							ClassGroupId: "bg-attachment",
						},
						"clip": {
							NextPart: map[string]ClassPart{
								"border": {
									ClassGroupId: "bg-clip",
								},
								"padding": {
									ClassGroupId: "bg-clip",
								},
								"content": {
									ClassGroupId: "bg-clip",
								},
								"text": {
									ClassGroupId: "bg-clip",
								},
							},
						},
						"opacity": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "bg-opacity",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "bg-opacity",
								},
							},
						},
						"origin": {
							NextPart: map[string]ClassPart{
								"border": {
									ClassGroupId: "bg-origin",
								},
								"padding": {
									ClassGroupId: "bg-origin",
								},
								"content": {
									ClassGroupId: "bg-origin",
								},
							},
						},
						"bottom": {
							ClassGroupId: "bg-position",
						},
						"center": {
							ClassGroupId: "bg-position",
						},
						"left": {
							NextPart: map[string]ClassPart{
								"bottom": {
									ClassGroupId: "bg-position",
								},
								"top": {
									ClassGroupId: "bg-position",
								},
							},
							ClassGroupId: "bg-position",
						},
						"right": {
							NextPart: map[string]ClassPart{
								"bottom": {
									ClassGroupId: "bg-position",
								},
								"top": {
									ClassGroupId: "bg-position",
								},
							},
							ClassGroupId: "bg-position",
						},
						"top": {
							ClassGroupId: "bg-position",
						},
						"no": {
							NextPart: map[string]ClassPart{
								"repeat": {
									ClassGroupId: "bg-repeat",
								},
							},
						},
						"repeat": {
							NextPart: map[string]ClassPart{
								"x": {
									ClassGroupId: "bg-repeat",
								},
								"y": {
									ClassGroupId: "bg-repeat",
								},
								"round": {
									ClassGroupId: "bg-repeat",
								},
								"space": {
									ClassGroupId: "bg-repeat",
								},
							},
							ClassGroupId: "bg-repeat",
						},
						"auto": {
							ClassGroupId: "bg-size",
						},
						"cover": {
							ClassGroupId: "bg-size",
						},
						"contain": {
							ClassGroupId: "bg-size",
						},
						"none": {
							ClassGroupId: "bg-image",
						},
						"gradient": {
							NextPart: map[string]ClassPart{
								"to": {
									NextPart: map[string]ClassPart{
										"t": {
											ClassGroupId: "bg-image",
										},
										"tr": {
											ClassGroupId: "bg-image",
										},
										"r": {
											ClassGroupId: "bg-image",
										},
										"br": {
											ClassGroupId: "bg-image",
										},
										"b": {
											ClassGroupId: "bg-image",
										},
										"bl": {
											ClassGroupId: "bg-image",
										},
										"l": {
											ClassGroupId: "bg-image",
										},
										"tl": {
											ClassGroupId: "bg-image",
										},
									},
								},
							},
						},
						"blend": {
							NextPart: map[string]ClassPart{
								"normal": {
									ClassGroupId: "bg-blend",
								},
								"multiply": {
									ClassGroupId: "bg-blend",
								},
								"screen": {
									ClassGroupId: "bg-blend",
								},
								"overlay": {
									ClassGroupId: "bg-blend",
								},
								"darken": {
									ClassGroupId: "bg-blend",
								},
								"lighten": {
									ClassGroupId: "bg-blend",
								},
								"color": {
									NextPart: map[string]ClassPart{
										"dodge": {
											ClassGroupId: "bg-blend",
										},
										"burn": {
											ClassGroupId: "bg-blend",
										},
									},
								},
								"hard": {
									NextPart: map[string]ClassPart{
										"light": {
											ClassGroupId: "bg-blend",
										},
									},
								},
								"soft": {
									NextPart: map[string]ClassPart{
										"light": {
											ClassGroupId: "bg-blend",
										},
									},
								},
								"difference": {
									ClassGroupId: "bg-blend",
								},
								"exclusion": {
									ClassGroupId: "bg-blend",
								},
								"hue": {
									ClassGroupId: "bg-blend",
								},
								"saturation": {
									ClassGroupId: "bg-blend",
								},
								"luminosity": {
									ClassGroupId: "bg-blend",
								},
								"plus": {
									NextPart: map[string]ClassPart{
										"lighter": {
											ClassGroupId: "bg-blend",
										},
									},
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryPosition,
							ClassGroupId: "bg-position",
						},
						{
							Fn:           IsArbitrarySize,
							ClassGroupId: "bg-size",
						},
						{
							Fn:           IsArbitraryImage,
							ClassGroupId: "bg-image",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "bg-color",
						},
					},
				},
				"from": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsPercent,
							ClassGroupId: "gradient-from-pos",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "gradient-from-pos",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "gradient-from",
						},
					},
				},
				"via": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsPercent,
							ClassGroupId: "gradient-via-pos",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "gradient-via-pos",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "gradient-via",
						},
					},
				},
				"to": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsPercent,
							ClassGroupId: "gradient-to-pos",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "gradient-to-pos",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "gradient-to",
						},
					},
				},
				"rounded": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "rounded",
						},
						"full": {
							ClassGroupId: "rounded",
						},
						"s": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-s",
								},
								"full": {
									ClassGroupId: "rounded-s",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-s",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-s",
								},
							},
							ClassGroupId: "rounded-s",
						},
						"e": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-e",
								},
								"full": {
									ClassGroupId: "rounded-e",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-e",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-e",
								},
							},
							ClassGroupId: "rounded-e",
						},
						"t": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-t",
								},
								"full": {
									ClassGroupId: "rounded-t",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-t",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-t",
								},
							},
							ClassGroupId: "rounded-t",
						},
						"r": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-r",
								},
								"full": {
									ClassGroupId: "rounded-r",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-r",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-r",
								},
							},
							ClassGroupId: "rounded-r",
						},
						"b": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-b",
								},
								"full": {
									ClassGroupId: "rounded-b",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-b",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-b",
								},
							},
							ClassGroupId: "rounded-b",
						},
						"l": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-l",
								},
								"full": {
									ClassGroupId: "rounded-l",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-l",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-l",
								},
							},
							ClassGroupId: "rounded-l",
						},
						"ss": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-ss",
								},
								"full": {
									ClassGroupId: "rounded-ss",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-ss",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-ss",
								},
							},
							ClassGroupId: "rounded-ss",
						},
						"se": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-se",
								},
								"full": {
									ClassGroupId: "rounded-se",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-se",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-se",
								},
							},
							ClassGroupId: "rounded-se",
						},
						"ee": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-ee",
								},
								"full": {
									ClassGroupId: "rounded-ee",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-ee",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-ee",
								},
							},
							ClassGroupId: "rounded-ee",
						},
						"es": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-es",
								},
								"full": {
									ClassGroupId: "rounded-es",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-es",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-es",
								},
							},
							ClassGroupId: "rounded-es",
						},
						"tl": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-tl",
								},
								"full": {
									ClassGroupId: "rounded-tl",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-tl",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-tl",
								},
							},
							ClassGroupId: "rounded-tl",
						},
						"tr": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-tr",
								},
								"full": {
									ClassGroupId: "rounded-tr",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-tr",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-tr",
								},
							},
							ClassGroupId: "rounded-tr",
						},
						"br": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-br",
								},
								"full": {
									ClassGroupId: "rounded-br",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-br",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-br",
								},
							},
							ClassGroupId: "rounded-br",
						},
						"bl": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "rounded-bl",
								},
								"full": {
									ClassGroupId: "rounded-bl",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "rounded-bl",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "rounded-bl",
								},
							},
							ClassGroupId: "rounded-bl",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsTshirtSize,
							ClassGroupId: "rounded",
						},
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "rounded",
						},
					},
					ClassGroupId: "rounded",
				},
				"border": {
					NextPart: map[string]ClassPart{
						"x": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "border-w-x",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "border-w-x",
								},
								{
									Fn:           IsAny,
									ClassGroupId: "border-color-x",
								},
							},
							ClassGroupId: "border-w-x",
						},
						"y": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "border-w-y",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "border-w-y",
								},
								{
									Fn:           IsAny,
									ClassGroupId: "border-color-y",
								},
							},
							ClassGroupId: "border-w-y",
						},
						"s": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "border-w-s",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "border-w-s",
								},
							},
							ClassGroupId: "border-w-s",
						},
						"e": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "border-w-e",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "border-w-e",
								},
							},
							ClassGroupId: "border-w-e",
						},
						"t": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "border-w-t",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "border-w-t",
								},
								{
									Fn:           IsAny,
									ClassGroupId: "border-color-t",
								},
							},
							ClassGroupId: "border-w-t",
						},
						"r": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "border-w-r",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "border-w-r",
								},
								{
									Fn:           IsAny,
									ClassGroupId: "border-color-r",
								},
							},
							ClassGroupId: "border-w-r",
						},
						"b": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "border-w-b",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "border-w-b",
								},
								{
									Fn:           IsAny,
									ClassGroupId: "border-color-b",
								},
							},
							ClassGroupId: "border-w-b",
						},
						"l": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "border-w-l",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "border-w-l",
								},
								{
									Fn:           IsAny,
									ClassGroupId: "border-color-l",
								},
							},
							ClassGroupId: "border-w-l",
						},
						"opacity": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "border-opacity",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "border-opacity",
								},
							},
							ClassGroupId: "border-opacity",
						},
						"solid": {
							ClassGroupId: "border-style",
						},
						"dashed": {
							ClassGroupId: "border-style",
						},
						"dotted": {
							ClassGroupId: "border-style",
						},
						"double": {
							ClassGroupId: "border-style",
						},
						"none": {
							ClassGroupId: "border-style",
						},
						"hidden": {
							ClassGroupId: "border-style",
						},
						"collapse": {
							ClassGroupId: "border-collapse",
						},
						"separate": {
							ClassGroupId: "border-collapse",
						},
						"spacing": {
							NextPart: map[string]ClassPart{
								"x": {
									Validators: []ClassGroupValidator{
										{
											Fn:           IsArbitraryValue,
											ClassGroupId: "border-spacing-x",
										},
										{
											Fn:           IsLength,
											ClassGroupId: "border-spacing-x",
										},
										{
											Fn:           IsArbitraryLength,
											ClassGroupId: "border-spacing-x",
										},
									},
								},
								"y": {
									Validators: []ClassGroupValidator{
										{
											Fn:           IsArbitraryValue,
											ClassGroupId: "border-spacing-y",
										},
										{
											Fn:           IsLength,
											ClassGroupId: "border-spacing-y",
										},
										{
											Fn:           IsArbitraryLength,
											ClassGroupId: "border-spacing-y",
										},
									},
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "border-spacing",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "border-spacing",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "border-spacing",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsLength,
							ClassGroupId: "border-w",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "border-w",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "border-color",
						},
					},
					ClassGroupId: "border-w",
				},
				"divide": {
					NextPart: map[string]ClassPart{
						"x": {
							NextPart: map[string]ClassPart{
								"reverse": {
									ClassGroupId: "divide-x-reverse",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "divide-x",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "divide-x",
								},
							},
							ClassGroupId: "divide-x",
						},
						"y": {
							NextPart: map[string]ClassPart{
								"reverse": {
									ClassGroupId: "divide-y-reverse",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "divide-y",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "divide-y",
								},
							},
							ClassGroupId: "divide-y",
						},
						"opacity": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "divide-opacity",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "divide-opacity",
								},
							},
						},
						"solid": {
							ClassGroupId: "divide-style",
						},
						"dashed": {
							ClassGroupId: "divide-style",
						},
						"dotted": {
							ClassGroupId: "divide-style",
						},
						"double": {
							ClassGroupId: "divide-style",
						},
						"none": {
							ClassGroupId: "divide-style",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsAny,
							ClassGroupId: "divide-color",
						},
					},
				},
				"outline": {
					NextPart: map[string]ClassPart{
						"solid": {
							ClassGroupId: "outline-style",
						},
						"dashed": {
							ClassGroupId: "outline-style",
						},
						"dotted": {
							ClassGroupId: "outline-style",
						},
						"double": {
							ClassGroupId: "outline-style",
						},
						"none": {
							ClassGroupId: "outline-style",
						},
						"offset": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "outline-offset",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "outline-offset",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsLength,
							ClassGroupId: "outline-w",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "outline-w",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "outline-color",
						},
					},
					ClassGroupId: "outline-style",
				},
				"ring": {
					NextPart: map[string]ClassPart{
						"inset": {
							ClassGroupId: "ring-w-inset",
						},
						"opacity": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "ring-opacity",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "ring-opacity",
								},
							},
						},
						"offset": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsLength,
									ClassGroupId: "ring-offset-w",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "ring-offset-w",
								},
								{
									Fn:           IsAny,
									ClassGroupId: "ring-offset-color",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsLength,
							ClassGroupId: "ring-w",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "ring-w",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "ring-color",
						},
					},
					ClassGroupId: "ring-w",
				},

				"shadow": {
					NextPart: map[string]ClassPart{
						"inner": {
							ClassGroupId: "shadow",
						},
						"none": {
							ClassGroupId: "shadow",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsTshirtSize,
							ClassGroupId: "shadow",
						},
						{
							Fn:           IsArbitraryShadow,
							ClassGroupId: "shadow",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "shadow-color",
						},
					},
					ClassGroupId: "shadow",
				},
				"opacity": {
					NextPart: map[string]ClassPart{},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsNumber,
							ClassGroupId: "opacity",
						},
						{
							Fn:           IsArbitraryNumber,
							ClassGroupId: "opacity",
						},
					},
				},

				"mix": {
					NextPart: map[string]ClassPart{
						"blend": {
							NextPart: map[string]ClassPart{
								"normal": {
									ClassGroupId: "mix-blend",
								},
								"multiply": {
									ClassGroupId: "mix-blend",
								},
								"screen": {
									ClassGroupId: "mix-blend",
								},
								"overlay": {
									ClassGroupId: "mix-blend",
								},
								"darken": {
									ClassGroupId: "mix-blend",
								},
								"lighten": {
									ClassGroupId: "mix-blend",
								},
								"color": {
									NextPart: map[string]ClassPart{
										"dodge": {
											ClassGroupId: "mix-blend",
										},
										"burn": {
											ClassGroupId: "mix-blend",
										},
									},
									ClassGroupId: "mix-blend",
								},
								"hard": {
									NextPart: map[string]ClassPart{
										"light": {
											ClassGroupId: "mix-blend",
										},
									},
								},
								"soft": {
									NextPart: map[string]ClassPart{
										"light": {
											ClassGroupId: "mix-blend",
										},
									},
								},
								"difference": {
									ClassGroupId: "mix-blend",
								},
								"exclusion": {
									ClassGroupId: "mix-blend",
								},
								"hue": {
									ClassGroupId: "mix-blend",
								},
								"saturation": {
									ClassGroupId: "mix-blend",
								},
								"luminosity": {
									ClassGroupId: "mix-blend",
								},
								"plus": {
									NextPart: map[string]ClassPart{
										"lighter": {
											ClassGroupId: "mix-blend",
										},
									},
								},
							},
						},
					},
				},
				"filter": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "filter",
						},
					},
					ClassGroupId: "filter",
				},
				"blur": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "blur",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsTshirtSize,
							ClassGroupId: "blur",
						},
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "blur",
						},
					},
					ClassGroupId: "blur",
				},

				"brightness": {
					NextPart: map[string]ClassPart{},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsNumber,
							ClassGroupId: "brightness",
						},
						{
							Fn:           IsArbitraryNumber,
							ClassGroupId: "brightness",
						},
					},
					ClassGroupId: "brightness",
				},

				"contrast": {
					NextPart: map[string]ClassPart{},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsNumber,
							ClassGroupId: "contrast",
						},
						{
							Fn:           IsArbitraryNumber,
							ClassGroupId: "contrast",
						},
					},
					ClassGroupId: "contrast",
				},

				"drop": {
					NextPart: map[string]ClassPart{
						"shadow": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "drop-shadow",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "drop-shadow",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "drop-shadow",
								},
							},
							ClassGroupId: "drop-shadow",
						},
					},
				},

				"grayscale": {
					NextPart: map[string]ClassPart{
						"0": {
							ClassGroupId: "grayscale",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "grayscale",
						},
					},
					ClassGroupId: "grayscale",
				},

				"hue": {
					NextPart: map[string]ClassPart{
						"rotate": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "hue-rotate",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "hue-rotate",
								},
							},
						},
					},
				},
				"invert": {
					NextPart: map[string]ClassPart{
						"0": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "invert",
								},
							},
							ClassGroupId: "invert",
						},
					},
					ClassGroupId: "invert",
				},

				"saturate": {
					NextPart: map[string]ClassPart{},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsNumber,
							ClassGroupId: "saturate",
						},
						{
							Fn:           IsArbitraryNumber,
							ClassGroupId: "saturate",
						},
					},
					ClassGroupId: "saturate",
				},

				"sepia": {
					NextPart: map[string]ClassPart{
						"0": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "sepia",
								},
							},
							ClassGroupId: "sepia",
						},
					},
					ClassGroupId: "sepia",
				},

				"backdrop": {
					NextPart: map[string]ClassPart{
						"filter": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "backdrop-filter",
								},
							},
							ClassGroupId: "backdrop-filter",
						},
						"blur": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "backdrop-blur",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsTshirtSize,
									ClassGroupId: "backdrop-blur",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "backdrop-blur",
								},
							},
							ClassGroupId: "backdrop-blur",
						},
						"brightness": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "backdrop-brightness",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "backdrop-brightness",
								},
							},
							ClassGroupId: "backdrop-brightness",
						},
						"contrast": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "backdrop-contrast",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "backdrop-contrast",
								},
							},
							ClassGroupId: "backdrop-contrast",
						},
						"grayscale": {
							NextPart: map[string]ClassPart{
								"0": {
									ClassGroupId: "backdrop-grayscale",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "backdrop-grayscale",
								},
							},
							ClassGroupId: "backdrop-grayscale",
						},
						"hue": {
							NextPart: map[string]ClassPart{
								"rotate": {
									Validators: []ClassGroupValidator{
										{
											Fn:           IsNumber,
											ClassGroupId: "backdrop-hue-rotate",
										},
										{
											Fn:           IsArbitraryValue,
											ClassGroupId: "backdrop-hue-rotate",
										},
									},
								},
							},
							Validators: []ClassGroupValidator{},
						},
						"invert": {
							NextPart: map[string]ClassPart{
								"0": {
									Validators: []ClassGroupValidator{
										{
											Fn:           IsArbitraryValue,
											ClassGroupId: "backdrop-invert",
										},
									},
									ClassGroupId: "backdrop-invert",
								},
							},
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "backdrop-invert",
						},
						"opacity": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "backdrop-opacity",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "backdrop-opacity",
								},
							},
							ClassGroupId: "backdrop-opacity",
						},
						"saturate": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "backdrop-saturate",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "backdrop-saturate",
								},
							},
							ClassGroupId: "backdrop-saturate",
						},
						"sepia": {
							NextPart: map[string]ClassPart{
								"0": {
									Validators: []ClassGroupValidator{
										{
											Fn:           IsArbitraryValue,
											ClassGroupId: "backdrop-sepia",
										},
									},
									ClassGroupId: "backdrop-sepia",
								},
							},
							ClassGroupId: "backdrop-sepia",
						},
					},
				},
				"caption": {
					NextPart: map[string]ClassPart{
						"top": {
							ClassGroupId: "caption",
						},
						"bottom": {
							ClassGroupId: "caption",
						},
					},
					Validators: []ClassGroupValidator{},
				},
				"transition": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "transition",
						},
						"all": {
							ClassGroupId: "transition",
						},
						"colors": {
							ClassGroupId: "transition",
						},
						"opacity": {
							ClassGroupId: "transition",
						},
						"shadow": {
							ClassGroupId: "transition",
						},
						"transform": {
							ClassGroupId: "transition",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "transition",
						},
					},
					ClassGroupId: "transition",
				},
				"duration": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsNumber,
							ClassGroupId: "duration",
						},
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "duration",
						},
					},
				},
				"ease": {
					NextPart: map[string]ClassPart{
						"linear": {
							ClassGroupId: "ease",
						},
						"in": {
							NextPart: map[string]ClassPart{
								"out": {
									ClassGroupId: "ease",
								},
							},
							ClassGroupId: "ease",
						},
						"out": {
							ClassGroupId: "ease",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "ease",
						},
					},
				},
				"delay": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsNumber,
							ClassGroupId: "delay",
						},
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "delay",
						},
					},
				},
				"animate": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "animate",
						},
						"spin": {
							ClassGroupId: "animate",
						},
						"ping": {
							ClassGroupId: "animate",
						},
						"pulse": {
							ClassGroupId: "animate",
						},
						"bounce": {
							ClassGroupId: "animate",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "animate",
						},
					},
				},
				"transform": {
					NextPart: map[string]ClassPart{
						"gpu": {
							ClassGroupId: "transform",
						},
						"none": {
							ClassGroupId: "transform",
						},
					},
					ClassGroupId: "transform",
				},
				"scale": {
					NextPart: map[string]ClassPart{
						"x": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "scale-x",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "scale-x",
								},
							},
						},
						"y": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "scale-y",
								},
								{
									Fn:           IsArbitraryNumber,
									ClassGroupId: "scale-y",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsNumber,
							ClassGroupId: "scale",
						},
						{
							Fn:           IsArbitraryNumber,
							ClassGroupId: "scale",
						},
					},
				},
				"rotate": {
					Validators: []ClassGroupValidator{
						{
							Fn:           IsInteger,
							ClassGroupId: "rotate",
						},
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "rotate",
						},
					},
				},
				"translate": {
					NextPart: map[string]ClassPart{
						"x": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "translate-x",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "translate-x",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "translate-x",
								},
							},
						},
						"y": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "translate-y",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "translate-y",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "translate-y",
								},
							},
						},
					},
				},
				"skew": {
					NextPart: map[string]ClassPart{
						"x": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "skew-x",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "skew-x",
								},
							},
						},
						"y": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsNumber,
									ClassGroupId: "skew-y",
								},
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "skew-y",
								},
							},
						},
					},
				},
				"origin": {
					NextPart: map[string]ClassPart{
						"center": {
							ClassGroupId: "transform-origin",
						},
						"top": {
							NextPart: map[string]ClassPart{
								"right": {
									ClassGroupId: "transform-origin",
								},
								"left": {
									ClassGroupId: "transform-origin",
								},
							},
							ClassGroupId: "transform-origin",
						},
						"right": {
							ClassGroupId: "transform-origin",
						},
						"bottom": {
							NextPart: map[string]ClassPart{
								"right": {
									ClassGroupId: "transform-origin",
								},
								"left": {
									ClassGroupId: "transform-origin",
								},
							},
							ClassGroupId: "transform-origin",
						},
						"left": {
							ClassGroupId: "transform-origin",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "transform-origin",
						},
					},
				},
				"accent": {
					NextPart: map[string]ClassPart{
						"auto": {
							NextPart:     map[string]ClassPart{},
							Validators:   []ClassGroupValidator{},
							ClassGroupId: "accent",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsAny,
							ClassGroupId: "accent",
						},
					},
				},
				"appearance": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "appearance",
						},
						"auto": {
							ClassGroupId: "appearance",
						},
					},
				},
				"cursor": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "cursor",
						},
						"default": {
							ClassGroupId: "cursor",
						},
						"pointer": {
							ClassGroupId: "cursor",
						},
						"wait": {
							ClassGroupId: "cursor",
						},
						"text": {
							ClassGroupId: "cursor",
						},
						"move": {
							ClassGroupId: "cursor",
						},
						"help": {
							ClassGroupId: "cursor",
						},
						"not": {
							NextPart: map[string]ClassPart{
								"allowed": {
									ClassGroupId: "cursor",
								},
							},
						},
						"none": {
							ClassGroupId: "cursor",
						},
						"context": {
							NextPart: map[string]ClassPart{
								"menu": {
									ClassGroupId: "cursor",
								},
							},
						},
						"progress": {
							ClassGroupId: "cursor",
						},
						"cell": {
							ClassGroupId: "cursor",
						},
						"crosshair": {
							ClassGroupId: "cursor",
						},
						"vertical": {
							NextPart: map[string]ClassPart{
								"text": {
									ClassGroupId: "cursor",
								},
							},
						},
						"alias": {
							ClassGroupId: "cursor",
						},
						"copy": {
							ClassGroupId: "cursor",
						},
						"no": {
							NextPart: map[string]ClassPart{
								"drop": {
									ClassGroupId: "cursor",
								},
							},
						},
						"grab": {
							ClassGroupId: "cursor",
						},
						"grabbing": {
							ClassGroupId: "cursor",
						},
						"all": {
							NextPart: map[string]ClassPart{
								"scroll": {
									ClassGroupId: "cursor",
								},
							},
						},
						"col": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"row": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"n": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"e": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"s": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"w": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"ne": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"nw": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"se": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"sw": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"ew": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"ns": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"nesw": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"nwse": {
							NextPart: map[string]ClassPart{
								"resize": {
									ClassGroupId: "cursor",
								},
							},
						},
						"zoom": {
							NextPart: map[string]ClassPart{
								"in": {
									ClassGroupId: "cursor",
								},
								"out": {
									ClassGroupId: "cursor",
								},
							},
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsArbitraryValue,
							ClassGroupId: "cursor",
						},
					},
				},
				"caret": {
					NextPart: map[string]ClassPart{},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsAny,
							ClassGroupId: "caret-color",
						},
					},
				},
				"pointer": {
					NextPart: map[string]ClassPart{
						"events": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "pointer-events",
								},
								"auto": {
									ClassGroupId: "pointer-events",
								},
							},
						},
					},
				},
				"resize": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "resize",
						},
						"y": {
							ClassGroupId: "resize",
						},
						"x": {
							ClassGroupId: "resize",
						},
					},
					ClassGroupId: "resize",
				},
				"scroll": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "scroll-behavior",
						},
						"smooth": {
							ClassGroupId: "scroll-behavior",
						},
						"m": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-m",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-m",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-m",
								},
							},
						},
						"mx": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-mx",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-mx",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-mx",
								},
							},
						},
						"my": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-my",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-my",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-my",
								},
							},
						},
						"ms": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-ms",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-ms",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-ms",
								},
							},
						},
						"me": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-me",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-me",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-me",
								},
							},
						},
						"mt": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-mt",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-mt",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-mt",
								},
							},
						},
						"mr": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-mr",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-mr",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-mr",
								},
							},
						},
						"mb": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-mb",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-mb",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-mb",
								},
							},
						},
						"ml": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-ml",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-ml",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-ml",
								},
							},
						},
						"p": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-p",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-p",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-p",
								},
							},
						},
						"px": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-px",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-px",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-px",
								},
							},
						},
						"py": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-py",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-py",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-py",
								},
							},
						},
						"ps": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-ps",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-ps",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-ps",
								},
							},
						},
						"pe": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-pe",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-pe",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-pe",
								},
							},
						},
						"pt": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-pt",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-pt",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-pt",
								},
							},
						},
						"pr": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-pr",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-pr",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-pr",
								},
							},
						},
						"pb": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-pb",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-pb",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-pb",
								},
							},
						},
						"pl": {
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "scroll-pl",
								},
								{
									Fn:           IsLength,
									ClassGroupId: "scroll-pl",
								},
								{
									Fn:           IsArbitraryLength,
									ClassGroupId: "scroll-pl",
								},
							},
						},
					},
				},
				"snap": {
					NextPart: map[string]ClassPart{
						"start": {
							ClassGroupId: "snap-align",
						},
						"end": {
							ClassGroupId: "snap-align",
						},
						"center": {
							ClassGroupId: "snap-align",
						},
						"align": {
							NextPart: map[string]ClassPart{
								"none": {
									ClassGroupId: "snap-align",
								},
							},
						},
						"normal": {
							ClassGroupId: "snap-stop",
						},
						"always": {
							ClassGroupId: "snap-stop",
						},
						"none": {
							ClassGroupId: "snap-type",
						},
						"x": {
							ClassGroupId: "snap-type",
						},
						"y": {
							ClassGroupId: "snap-type",
						},
						"both": {
							ClassGroupId: "snap-type",
						},
						"mandatory": {
							ClassGroupId: "snap-strictness",
						},
						"proximity": {
							ClassGroupId: "snap-strictness",
						},
					},
				},
				"touch": {
					NextPart: map[string]ClassPart{
						"auto": {
							ClassGroupId: "touch",
						},
						"none": {
							ClassGroupId: "touch",
						},
						"manipulation": {
							ClassGroupId: "touch",
						},
						"pan": {
							NextPart: map[string]ClassPart{
								"x": {
									ClassGroupId: "touch-x",
								},
								"left": {
									ClassGroupId: "touch-x",
								},
								"right": {
									ClassGroupId: "touch-x",
								},
								"y": {
									ClassGroupId: "touch-y",
								},
								"up": {
									ClassGroupId: "touch-y",
								},
								"down": {
									ClassGroupId: "touch-y",
								},
							},
						},
						"pinch": {
							NextPart: map[string]ClassPart{
								"zoom": {
									ClassGroupId: "touch-pz",
								},
							},
						},
					},
				},
				"select": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "select",
						},
						"text": {
							ClassGroupId: "select",
						},
						"all": {
							ClassGroupId: "select",
						},
						"auto": {
							ClassGroupId: "select",
						},
					},
					Validators: []ClassGroupValidator{},
				},
				"will": {
					NextPart: map[string]ClassPart{
						"change": {
							NextPart: map[string]ClassPart{
								"auto": {
									ClassGroupId: "will-change",
								},
								"scroll": {
									ClassGroupId: "will-change",
								},
								"contents": {
									ClassGroupId: "will-change",
								},
								"transform": {
									ClassGroupId: "will-change",
								},
							},
							Validators: []ClassGroupValidator{
								{
									Fn:           IsArbitraryValue,
									ClassGroupId: "will-change",
								},
							},
						},
					},
				},
				"fill": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "fill",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsAny,
							ClassGroupId: "fill",
						},
					},
				},
				"stroke": {
					NextPart: map[string]ClassPart{
						"none": {
							ClassGroupId: "stroke",
						},
					},
					Validators: []ClassGroupValidator{
						{
							Fn:           IsLength,
							ClassGroupId: "stroke-w",
						},
						{
							Fn:           IsArbitraryLength,
							ClassGroupId: "stroke-w",
						},
						{
							Fn:           IsArbitraryNumber,
							ClassGroupId: "stroke-w",
						},
						{
							Fn:           IsAny,
							ClassGroupId: "stroke",
						},
					},
				},
				"sr": {
					NextPart: map[string]ClassPart{
						"only": {
							ClassGroupId: "sr",
						},
					},
				},
				"forced": {
					NextPart: map[string]ClassPart{
						"color": {
							NextPart: map[string]ClassPart{
								"adjust": {
									NextPart: map[string]ClassPart{
										"auto": {
											ClassGroupId: "forced-color-adjust",
										},
										"none": {
											ClassGroupId: "forced-color-adjust",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
