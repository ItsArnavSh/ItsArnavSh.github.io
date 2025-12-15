<script lang="ts">
	import Projectsbutton from '../components/projectsbutton.svelte';
	import Resume from '../components/resume.svelte';
	import './layout.css';
</script>

<div class="bg-(--darkbg) h-lvh w-lvw">
	<div class="flex flex-row pl-5 pt-30">
		<div class="text-(--accent) text-9xl bayon">ARNAV</div>
		<div class="text-(--silver) share-tech pl-3 mt-5">
			Professional<br />Wheel Reinventor <br />and Overthinker
		</div>
	</div>
	<p class="text-(--lightbg) pl-5 pt-10 text-4xl share-tech max-w-[50%]">
		someone who likes following curiosity wherever it leads.<br />I spend my time building n
		breaking things, and chasing ideas that make life feel a little bigger.<br />
		Whether it’s a new project, a book, or a strange thought at 2 AM, I enjoy exploring the world with
		a mix of wonder, stubbornness, and humor.
	</p>
	<div class="flex flex-row">
		<p class="text-(--lightbg) pl-5 pt-10 text-4xl share-tech max-w-[50%]">
			If you vibe with it, check out
		</p>

		<p
			class="text-(--lightbg) pl-5 pt-8 text-5xl bayon max-w-[50%]
           "
		>
			THE
			<a
				href="/blog"
				class="text-(--accent) drop-shadow-[8px_6px_0px_rgba(0,0,0,1)] hover:drop-shadow-[15px_12px_0px_rgba(0,0,0,1)] transition-all duration-300"
			>
				BLOG
			</a>
		</p>
	</div>
	<p class="text-(--lightbg) pl-5 pt-10 text-4xl share-tech max-w-[50%]">
		..-. ..- -.-. -.- / .--- ...
	</p>
	<div class="flex flex-row">
		<div class="mt-10 pl-5 w-[70%]">
			<Projectsbutton />
		</div>

		<Resume />
	</div>
</div>
