<script>
	import { onMount } from 'svelte';

	let projects = [];
	let filteredProjects = [];
	let loading = true;
	let error = null;
	let searchQuery = '';

	onMount(async () => {
		try {
			const response = await fetch(
				'https://api.github.com/repos/ItsArnavSh/ItsArnavSh.github.io/contents/data/projects.json'
			);
			const data = await response.json();
			const content = JSON.parse(atob(data.content));

			// Sorting projects by last_updated in descending order
			projects = content.projects.sort((a, b) => {
				const [monthA, yearA] = a.last_updated.split('-').map(Number);
				const [monthB, yearB] = b.last_updated.split('-').map(Number);

				// Compare years first, then months if years are equal
				return yearB - yearA || monthB - monthA;
			});

			filteredProjects = [...projects];
		} catch (e) {
			error = 'Failed to load projects';
		} finally {
			loading = false;
		}
	});

	function searchProjects() {
		const query = searchQuery.toLowerCase();
		filteredProjects = projects.filter(
			(project) =>
				project.name.toLowerCase().includes(query) ||
				project.tags.some((tag) => tag.toLowerCase().includes(query))
		);
	}
</script>

<section id="projects" class="min-h-screen py-20">
	<div class="container mx-auto px-4">
		<h2 class="alkatra mb-12 text-center text-8xl">Projects</h2>

		<div class="mb-6 w-full">
			<input
				type="text"
				bind:value={searchQuery}
				on:input={searchProjects}
				placeholder="Search projects by name or tag..."
				class="w-full rounded-md border p-3 transition-all duration-300"
				style="min-height: 3rem;"
			/>
		</div>

		{#if loading}
			<div class="flex justify-center">
				<div class="h-12 w-12 animate-spin rounded-full border-b-2 border-[#9797F3]"></div>
			</div>
		{:else if error}
			<p class="text-center text-red-500">{error}</p>
		{:else}
			{#if filteredProjects.length === 0}
				<p class="text-center text-gray-500">No projects found.</p>
			{/if}
			<div class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
				{#each filteredProjects as project}
					<div
						class="fredoka overflow-hidden rounded-lg bg-white shadow-lg transition-shadow duration-300 hover:shadow-xl"
					>
						<div class="p-6">
							<div class="mb-4 flex items-start justify-between">
								<h3 class="text-xl font-semibold">{project.name}</h3>
								<span
									class="rounded-full px-3 py-1 text-sm"
									style="background-color: {project.status === 'Completed' ? '#E6FFF3' : '#FFF3E6'};
									color: {project.status === 'Completed' ? '#00875A' : '#FF8B00'}"
								>
									{project.status}
								</span>
							</div>
							<p class="mb-4 text-gray-600">{project.description}</p>
							<div class="mb-4 flex flex-wrap gap-2">
								{#each project.tags as tag}
									<span class="rounded-full bg-[#F0F0FF] px-3 py-1 text-sm text-[#9797F3]"
										>{tag}</span
									>
								{/each}
							</div>
							<div class="space-y-2">
								{#each project.features.slice(0, 2) as feature}
									<div class="flex items-center text-sm text-gray-600">
										<span class="mr-2">•</span>
										{feature}
									</div>
								{/each}
							</div>
							<div class="mt-6 flex items-center justify-between">
								<div class="flex items-center">
									<span class="text-sm text-gray-600">{project.language}</span>
								</div>
								<a
									href={project.repository}
									target="_blank"
									rel="noopener noreferrer"
									class="text-[#9797F3] transition-colors duration-300 hover:text-[#7777D3]"
								>
									View Project →
								</a>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</section>

<style>
	:global(.font-alexandria) {
		font-family: 'Alexandria', sans-serif;
	}
</style>
