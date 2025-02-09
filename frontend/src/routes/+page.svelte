<script>
	import { onMount } from 'svelte';
	import Hero from './Hero.svelte';
	import Projects from './Projects.svelte';
	import Blog from './Blog.svelte';
	import Newsletter from './Newsletter.svelte';
	let isScrolled = false;
	let activeSection = 'hero';

	onMount(() => {
		// Handle scroll for navbar background
		window.addEventListener('scroll', () => {
			isScrolled = window.scrollY > 20;

			// Update active section based on scroll position
			const sections = document.querySelectorAll('section');
			sections.forEach((section) => {
				const rect = section.getBoundingClientRect();
				if (rect.top <= 100 && rect.bottom >= 100) {
					activeSection = section.id;
				}
			});
		});
	});

	// Smooth scroll function
	function scrollToSection(sectionId) {
		const element = document.getElementById(sectionId);
		element?.scrollIntoView({ behavior: 'smooth' });
	}
</script>

<!-- Fixed Navbar -->
<nav
	class="fixed right-12 top-8 z-50 flex items-center gap-8 rounded-full px-8 py-4 transition-all duration-300"
	class:bg-white={isScrolled}
	class:shadow-lg={isScrolled}
>
	<ul class="flex items-center gap-8">
		{#each ['about', 'projects', 'blog', 'newsletter'] as section}
			<li>
				<button
					on:click={() => scrollToSection(section)}
					class="alexandria relative text-lg transition-colors duration-300"
					class:text-black={activeSection === section}
					class:text-gray-400={activeSection !== section}
				>
					{section.charAt(0).toUpperCase() + section.slice(1)}
					{#if activeSection === section}
						<div
							class="absolute -bottom-1 left-0 h-0.5 w-full bg-black transition-all duration-300"
						/>
					{/if}
				</button>
			</li>
		{/each}
	</ul>
</nav>

<main class="bg-white">
	<!-- Hero Section -->
	<section id="about" class="min-h-screen">
		<Hero />
	</section>

	<!-- Projects Section -->
	<section id="projects" class="flex min-h-screen w-full items-center justify-center">
		<Projects />
	</section>

	<!-- Blog Section -->
	<section id="blog" class="flex min-h-screen items-center justify-center">
		<Blog />
	</section>

	<!-- Newsletter Section -->
	<section id="newsletter" class="flex min-h-screen items-center justify-center">
		<Newsletter />
	</section>
</main>

<style>
	:global(body) {
		background-color: #faf8f6;
		scroll-behavior: smooth;
	}

	:global(.bg-coral-400) {
		background-color: #ff906d;
	}

	:global(.bg-coral-500) {
		background-color: #ff7f5d;
	}

	section {
		scroll-margin-top: 2rem;
	}

	nav {
		backdrop-filter: blur(8px);
		background-color: rgba(255, 255, 255, 0.8);
	}

	nav:not(.bg-white) {
		background-color: transparent;
	}
</style>
