<script lang="ts">
	import { onMount } from 'svelte';
	import Chart from 'chart.js/auto';

	let canvas: HTMLCanvasElement;
	let showDates = false;
	let count = 100;

	onMount(async () => {
		let res = await fetch('/api/results');
		let data: { destination: string; rtt: number; seq: number; at: number }[] = await res.json();

		let datasets: Record<string, { label: string; data: number[] }> = {};
		let datasetLebels = new Set(...[data.map((r) => r.destination)]);
		for (const l of datasetLebels) {
			datasets[l] = {
				label: l,
				data: data
					.filter((r) => r.destination === l)
					.map((r) => r.rtt)
					.slice(-count)
			};
		}

		const ch = new Chart(canvas, {
			type: 'line',
			options: {
				// responsive: true,
				maintainAspectRatio: false,
				plugins: {
					legend: {
						labels: {
							color: '#fafafa',
							font: {
								size: 16
							}
						}
					}
				},
				scales: {
					x: {
						ticks: {
							autoSkip: true,
							maxTicksLimit: 10,
							color: '#fafafa'
						},
						grid: {
							color: '#1c2541'
						}
					},
					y: {
						ticks: {
							color: '#fafafa'
						},
						grid: {
							color: '#1c2541'
						}
					}
				},
				elements: {
					point: {
						radius: 2
					},
					line: {
						tension: 0.2,
						borderJoinStyle: 'bevel',
						borderWidth: 2
					}
				}
			},
			data: {
				labels: data
					.map((r) => {
						const d = new Date(r.at);
						return (
							(showDates ? d.toDateString() : '') +
							` ${d.getHours()}:${d.getMinutes()}:${d.getSeconds()}`
						);
					})
					.slice(-count),
				datasets: Object.values(datasets)
			}
		});

		while (true) {
			let res = await fetch('/api/results');
			let data: { destination: string; rtt: number; seq: number; at: number }[] = await res.json();

			datasetLebels = new Set(...[data.map((r) => r.destination)]);
			for (const l of datasetLebels) {
				datasets[l] = {
					label: l,
					data: data
						.filter((r) => r.destination === l)
						.map((r) => r.rtt)
						.slice(-count)
				};
			}

			ch.data.labels = data
				.map((r) => {
					const d = new Date(r.at);
					return (
						(showDates ? d.toDateString() : '') +
						` ${d.getHours()}:${d.getMinutes()}:${d.getSeconds()}`
					);
				})
				.slice(-count);
			ch.data.datasets = Object.values(datasets);
			ch.update('resize');

			await new Promise((r) => setTimeout(r, 1000));
		}
	});
</script>

<div class="mb-4 flex items-center">
	<div class="flex items-center mr-4">
		<input id="show-dates" type="checkbox" class="mr-2" bind:value={showDates} />
		<label for="show-dates"> Show Dates </label>
	</div>
	<div class="flex items-center">
		<label for="count"> Count: </label>
		<select
			on:change={(e) => (count = Number(e.currentTarget.value))}
			id="count"
			class="ml-2 bg-[#0b132b]"
		>
			<option value="100">100</option>
			<option value="200">200</option>
			<option value="500">500</option>
		</select>
	</div>
</div>
<div>
	<canvas
		bind:this={canvas}
		class="w-full h-[calc(100svh-136px)] p-4 bg-[#0b132b] rounded shadow-sm"
	></canvas>
</div>
