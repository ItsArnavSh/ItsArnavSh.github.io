<script>
	import { onMount } from 'svelte';
	import { marked } from 'marked';

	let blogs = [];
	let filteredBlogs = [];
	let loading = true;
	let error = null;
	let searchQuery = '';
	let showModal = false;
	let currentBlog = { title: '', content: '' };

	onMount(async () => {
		try {
			const response = await fetch(
				'https://raw.githubusercontent.com/ItsArnavSh/ItsArnavSh.github.io/main/data/contents.json'
			);
			const data = await response.json();

			blogs = data.blogs.sort((a, b) => {
				const [dayA, monthA, yearA] = a.date_created.split('-').map(Number);
				const [dayB, monthB, yearB] = b.date_created.split('-').map(Number);
				return yearB - yearA || monthB - monthA || dayB - dayA;
			});

			filteredBlogs = [...blogs];
		} catch (e) {
			error = 'Failed to load blogs';
		} finally {
			loading = false;
		}
	});

	function searchBlogs() {
		const query = searchQuery.toLowerCase();
		filteredBlogs = blogs.filter(
			(blog) =>
				blog.file.toLowerCase().includes(query) || blog.summary.toLowerCase().includes(query)
		);
	}

	async function openBlog(blog) {
		try {
			const response = await fetch(
				`https://raw.githubusercontent.com/ItsArnavSh/ItsArnavSh.github.io/main/data/blog/${blog.file}`
			);
			const text = await response.text();
			currentBlog = { title: blog.summary, content: marked(text) };
			showModal = true;
		} catch (e) {
			alert('Failed to load the blog content');
		}
	}

	function closeModal() {
		showModal = false;
		currentBlog = { title: '', content: '' };
	}
</script>

<section class="min-h-screen w-full py-20">
	<div class="container mx-auto px-4">
		<h2 class="mb-12 text-center text-5xl font-bold">Blogs</h2>

		<div class="mb-6 w-full">
			<input
				type="text"
				bind:value={searchQuery}
				on:input={searchBlogs}
				placeholder="Search blogs..."
				class="w-full rounded-md border p-3 shadow-sm"
				style="min-height: 3rem;"
			/>
		</div>

		{#if loading}
			<div class="flex justify-center">
				<div class="h-12 w-12 animate-spin rounded-full border-b-2 border-blue-500"></div>
			</div>
		{:else if error}
			<p class="text-center text-red-500">{error}</p>
		{:else}
			{#if filteredBlogs.length === 0}
				<p class="text-center text-gray-500">No blogs found.</p>
			{/if}
			<div class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
				{#each filteredBlogs as blog}
					<button
						class="rounded-lg bg-white p-6 text-left shadow-md transition hover:shadow-lg"
						on:click={() => openBlog(blog)}
					>
						<h3 class="text-xl font-semibold">{blog.summary}</h3>
						<p class="mt-2 text-sm text-[#9797F3]">{blog.date_created}</p>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</section>

{#if showModal}
	<div class="fixed inset-0 z-20 flex w-full items-center justify-center bg-black bg-opacity-50">
		<div
			class="relative max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-white p-6 shadow-lg"
		>
			<button class="absolute right-4 top-4 text-gray-600 hover:text-black" on:click={closeModal}>
				✖
			</button>
			<h2 class="mb-4 text-2xl font-bold">{currentBlog.title}</h2>
			<div class="prose max-w-none">{@html currentBlog.content}</div>
		</div>
	</div>
{/if}
