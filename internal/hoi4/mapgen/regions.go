package mapgen

func regions(width, height int, same func(a, b int) bool) (labels []int32, sizes []int, edge []bool) {
	labels = make([]int32, width*height)
	for i := range labels {
		labels[i] = -1
	}
	queue := make([]int32, 0, 1024)
	for start := range labels {
		if labels[start] >= 0 {
			continue
		}
		label := int32(len(sizes))
		labels[start] = label
		queue = append(queue[:0], int32(start))
		size, touches := 0, false
		for len(queue) > 0 {
			p := int(queue[len(queue)-1])
			queue = queue[:len(queue)-1]
			size++
			x, y := p%width, p/width
			if x == 0 || y == 0 || x == width-1 || y == height-1 {
				touches = true
			}
			try := func(q int) {
				if labels[q] < 0 && same(p, q) {
					labels[q] = label
					queue = append(queue, int32(q))
				}
			}
			if x > 0 {
				try(p - 1)
			}
			if x < width-1 {
				try(p + 1)
			}
			if y > 0 {
				try(p - width)
			}
			if y < height-1 {
				try(p + width)
			}
		}
		sizes = append(sizes, size)
		edge = append(edge, touches)
	}
	return labels, sizes, edge
}

func clean(classes []Class, width, height, speck, lakeMax int) {
	removeSpecks(classes, width, height, speck)
	sortWater(classes, width, height, lakeMax)
}

func removeSpecks(classes []Class, width, height, speck int) {
	land := func(i int) bool { return classes[i].Land() }
	labels, sizes, _ := regions(width, height, func(a, b int) bool { return land(a) == land(b) })
	for i, label := range labels {
		if sizes[label] >= speck {
			continue
		}
		if land(i) {
			classes[i] = Sea
			continue
		}

		x, y := i%width, i/width
		filled := Plains
		for _, q := range [4][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
			if q[0] >= 0 && q[1] >= 0 && q[0] < width && q[1] < height && classes[q[1]*width+q[0]].Land() {
				filled = classes[q[1]*width+q[0]]
				break
			}
		}
		classes[i] = filled
	}
}

func sortWater(classes []Class, width, height, lakeMax int) {
	land := func(i int) bool { return classes[i].Land() }
	labels, sizes, edge := regions(width, height, func(a, b int) bool { return land(a) == land(b) })
	for i, label := range labels {
		if land(i) {
			continue
		}
		if !edge[label] && sizes[label] <= lakeMax {
			classes[i] = Lake
		} else {
			classes[i] = Sea
		}
	}
}
