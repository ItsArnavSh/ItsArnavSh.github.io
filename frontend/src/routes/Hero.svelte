<script lang="ts">
	import { fade, scale } from 'svelte/transition';

	const socials = [
		{ href: 'mailto:its.arnav.sh@gmail.com', label: 'Email', icon: 'Mail.png' },
		{ href: 'https://github.com/itsarnavsh', label: 'GitHub', icon: 'Github.png' },
		{ href: 'https://linkedin.com/in/itsarnavsh', label: 'LinkedIn', icon: 'Linkedin.png' }
	];
</script>

<div class="relative flex min-h-screen bg-white">
	<!-- Main Content -->
	<div class="flex w-full flex-col md:flex-row">
		<!-- Purple Left Side -->
		<div
			class="mb-10 mt-10 flex w-full flex-col items-center justify-center rounded-3xl bg-[#8B8FE5] p-6 sm:w-2/3 md:ml-10 md:mt-10 md:w-1/3 md:p-10"
		>
			<div class="flex flex-col items-center md:items-start">
				<img
					src="me.png"
					alt="Profile"
					class=" rounded-full object-cover md:relative md:left-32 md:h-max md:w-max"
				/>
				<!-- Left-Aligned "I am" -->
				<h1 class="text-center text-white md:text-left">
					<span class="inter block text-6xl tracking-wide md:text-8xl">I am</span>
					<span class="alkatra mt-3 block text-6xl md:text-9xl">Arnav</span>
					<p class="alkatra mb-3 mt-3 text-sm md:text-5xl">.--- ... / .... .- - . .-.</p>
				</h1>
			</div>
		</div>

		<!-- White Right Side -->
		<div class="flex w-full flex-col items-center justify-center p-8 md:w-2/3 md:p-12">
			<!-- Description -->
			<div class="space-y-6" in:fade={{ duration: 300, delay: 600 }}>
				<h2 class="fredoka mb-8 text-center text-4xl text-gray-800 md:text-6xl">
					CSE Undergrad at JIIT.
				</h2>
				<p class="fredoka space-y-8 text-center text-3xl md:text-5xl">
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
			<div class="mb-12 mt-10 flex flex-row flex-wrap justify-center gap-4 p-1">
				{#each socials as { href, icon, label }}
					<a
						{href}
						target="_blank"
						rel="noopener noreferrer"
						class="transition-transform duration-300 hover:scale-110"
					>
						<span class="sr-only">{label}</span>
						<img
							src={icon || '/placeholder.svg'}
							alt={label}
							class="h-6 w-6 sm:h-8 sm:w-8 md:h-12 md:w-12 lg:h-16 lg:w-16"
							style="max-width: 32px; max-height: 32px;"
						/>
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

	/* Responsive styles for mobile */
	@media (max-width: 768px) {
		/* Adjusting the profile image */
		img {
			width: 12rem; /* Smaller size for mobile */
			height: 12rem;
		}

		/* Title adjustments */
		h1 span {
			font-size: 2.5rem; /* Smaller title size */
		}

		/* Description text */
		p {
			font-size: 1.5rem; /* Smaller text for mobile */
		}
	}
</style>
