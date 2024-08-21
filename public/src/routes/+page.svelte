<script lang="ts">
	import { onMount } from 'svelte';
	import Chart from 'chart.js/auto';
	import protobuf from 'protobufjs';

	let canvas: HTMLCanvasElement;
	let showDates = false;
	let count = 100;

	onMount(async () => {
		const pb = await protobuf.load('/PingResults.proto');
		let res = await fetch('/api/results');
		let ab = await res.arrayBuffer();
		let data: { Destination: string; RTT: number; Seq: number; At: number }[] = pb
			.lookupType('PingResults')
			.decode(new Uint8Array(ab), ab.byteLength)
			.toJSON().Results;

		let datasets: Record<string, { label: string; data: number[] }> = {};
		let datasetLebels = new Set(...[data.map((r) => r.Destination)]);
		for (const l of datasetLebels) {
			datasets[l] = {
				label: l,
				data: data
					.filter((r) => r.Destination === l)
					.map((r) => r.RTT)
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
						const d = new Date(Number(r.At));
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
			const ab = await res.arrayBuffer();
			let data: { Destination: string; RTT: number; Seq: number; At: number }[] = pb
				.lookupType('PingResults')
				.decode(new Uint8Array(ab), ab.byteLength)
				.toJSON().Results;
			datasetLebels = new Set(...[data.map((r) => r.Destination)]);
			for (const l of datasetLebels) {
				datasets[l] = {
					label: l,
					data: data
						.filter((r) => r.Destination === l)
						.map((r) => r.RTT)
						.slice(-count)
				};
			}

			ch.data.labels = data
				.map((r) => {
					const d = new Date(Number(r.At));
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

<div class="flex items-center pb-4">
	<div class="flex items-center">
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
	<a target="_blank" href="https://github.com/alirezasn3/watchcat" class="ml-auto text-sm underline"
		>WatchCat</a
	>
</div>
<div>
	<canvas bind:this={canvas} class="w-full h-[calc(100svh-72px)] p-4 bg-[#0b132b] rounded shadow-sm"
	></canvas>
</div>

<svelte:head>
	<title>WatchCat</title>
</svelte:head>
