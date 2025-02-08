<script lang="ts">
	import { fade, scale } from 'svelte/transition';

	const navItems = [
		{ href: '/newsletter', label: 'Newsletter' },
		{ href: '/blog', label: 'Blog      ' },
		{ href: '/projects', label: 'Projects  ' }
	];

	const socials = [
		{ href: 'mailto:its.arnav.sh@gmail.com', label: 'Email', icon: 'Mail.png' },
		{ href: 'https://github.com/itsarnavsh', label: 'GitHub', icon: 'Github.png' },
		{ href: 'https://linkedin.com/in/itsarnavsh', label: 'LinkedIn', icon: 'Linkedin.png' }
	];
</script>

<div class="relative flex min-h-screen bg-white">
	<!-- Vertical Navigation -->
	<nav class="fixed left-4 top-1/2 z-10 flex -translate-y-1/2 flex-col gap-36">
		{#each navItems as { href, label }}
			<a
				{href}
				class="alexandria -rotate-90 transform whitespace-nowrap text-lg tracking-wide text-gray-700 transition-colors duration-300 hover:text-indigo-500"
			>
				{label}
			</a>
		{/each}
	</nav>

	<!-- Main Content -->
	<div class="flex w-full flex-col md:flex-row">
		<!-- Purple Left Side -->
		<div class="mb-10 ml-40 mt-10 flex w-1/2 rounded-xl bg-[#8B8FE5] md:p-20">
			<div class="flex flex-col">
				<img src="me.png" alt="Profile" class="relative left-40 h-auto w-full object-cover" />
				<!-- Left-Aligned "I am" -->
				<h1 class=" text-left text-white">
					<span class="inter relative right-[30] block text-8xl tracking-wide md:text-9xl"
						>I am</span
					>
					<span class="alkatra md:text-[10]xl relative bottom-5 mt-3 block text-right text-9xl"
						>Arnav</span
					>
				</h1>
			</div>
		</div>

		<!-- White Right Side -->
		<div class="flex w-1/2 flex-col items-center justify-center p-8 md:p-12">
			<!-- Description -->
			<div class="space-y-6" in:fade={{ duration: 300, delay: 600 }}>
				<h2 class="anta mb-8 text-5xl font-medium text-gray-800">CSE Undergrad at JIIT.</h2>
				<p class="anta space-y-4 text-5xl font-bold">
					<span class="inline-block">
						<span class="relative rounded px-2 py-1">
							<span class="relative z-10">Backend</span>
							<span class="absolute bottom-0 left-0 right-0 h-2/5 rounded bg-[#44FFC4]/50"></span>
						</span>
						<span class="mx-2">and</span>
						<span class="relative rounded px-2 py-1">
							<span class="relative z-10">Systems</span>
							<span class="absolute bottom-0 left-0 right-0 h-2/5 rounded bg-[#FF4A44]/50"></span>
						</span>
					</span>
					<br />
					<span class="inline-block">Developer, interested in</span>
					<br />
					<span class="inline-block">
						<span class="relative rounded px-2 py-1">
							<span class="relative z-10">Distributed Systems</span>
							<span class="absolute bottom-0 left-0 right-0 h-2/5 rounded bg-[#FFCD06]/50"></span>
						</span>
					</span>
					<span class="inline-block">and</span>
					<br />
					<span class="inline-block">
						<span class="relative rounded px-2 py-1">
							<span class="relative z-10">Quantum Computing</span>
							<span class="absolute bottom-0 left-0 right-0 h-2/5 rounded bg-[#B444FF]/40"></span>
						</span>
					</span>
				</p>
			</div>
			<!-- Social Links -->
			<div class="mb-12 flex flex-row gap-6 p-10">
				{#each socials as { href, icon, label }}
					<a
						{href}
						target="_blank"
						rel="noopener noreferrer"
						class="text-3xl text-gray-700 transition-transform duration-300 hover:scale-110"
						in:scale={{ duration: 300, delay: 300 }}
					>
						<span class="sr-only">{label}</span>
						<img src={icon} alt={label} />
					</a>
				{/each}
			</div>
		</div>
	</div>
</div>

<style>
	@import url('https://fonts.googleapis.com/css2?family=Playfair+Display:ital@1&display=swap');

	:global(body) {
		margin: 0;
		padding: 0;
	}
</style>
