<script lang="ts">
	import { onMount } from 'svelte';
	import Chart from 'chart.js/auto';

	let canvas: HTMLCanvasElement;

	onMount(async () => {
		let res = await fetch('http://49.13.53.236');
		let data: { destination: string; rtt: number; seq: number; at: number }[] = await res.json();

		let datasets: Record<string, { label: string; data: number[] }> = {};
		let datasetLebels = new Set(...[data.map((r) => r.destination)]);
		for (const l of datasetLebels) {
			datasets[l] = {
				label: l,
				data: data
					.filter((r) => r.destination === l)
					.map((r) => r.rtt)
					.slice(-100)
			};
		}

		const ch = new Chart(canvas, {
			type: 'line',
			options: {
				responsive: true,
				scales: {
					x: {
						ticks: {
							autoSkip: true,
							maxTicksLimit: 10
						}
					},
					y: {
						min: 50,
						max: 250
					}
				},
				elements: {
					point: {
						radius: 0
					},
					line: {
						tension: 0,
						borderJoinStyle: 'bevel',
						borderWidth: 2
					}
				}
			},
			data: {
				labels: data
					.map((r) => {
						const d = new Date(r.at);
						return d.toDateString() + ` ${d.getHours()}:${d.getMinutes()}:${d.getSeconds()}`;
					})
					.slice(-100),
				datasets: Object.values(datasets)
			}
		});

		while (true) {
			let res = await fetch('http://49.13.53.236');
			let data: { destination: string; rtt: number; seq: number; at: number }[] = await res.json();

			datasetLebels = new Set(...[data.map((r) => r.destination)]);
			for (const l of datasetLebels) {
				datasets[l] = {
					label: l,
					data: data
						.filter((r) => r.destination === l)
						.map((r) => r.rtt)
						.slice(-100)
				};
			}

			ch.data.labels = data
				.map((r) => {
					const d = new Date(r.at);
					return d.toDateString() + ` ${d.getHours()}:${d.getMinutes()}:${d.getSeconds()}`;
				})
				.slice(-100);
			ch.data.datasets = Object.values(datasets);
			ch.update('none');

			await new Promise((r) => setTimeout(r, 1000));
		}
	});
</script>

<!-- <div>
	{#each results as result}
		<div>
			{result.destination} - {result.seq} - {result.rtt}ms
		</div>
	{/each}
</div> -->

<div>
	<canvas bind:this={canvas}></canvas>
</div>
